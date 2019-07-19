package tcache

import (
	"errors"
	"time"

	"github.com/vmihailenco/msgpack"

	"github.com/go-redis/redis"
	"github.com/rs/zerolog/log"
)

var ErrKeyCacheNotFound = errors.New("cache: key not found")

const (
	tagPrefixCache = "tag:"
)

// Item - entity of the added object
type Item struct {
	Key        string
	Object     interface{}
	Expiration time.Duration
}

// TCache - main library structure
type TCache struct {
	// redis cache client
	redis *redis.Client

	// Processing / Post-processing methods
	Marshal   func(interface{}) ([]byte, error)
	Unmarshal func([]byte, interface{}) error
}

// NewTCache - constructor
func NewTCache(redis *redis.Client) *TCache {
	return &TCache{
		redis: redis,
		Marshal: func(v interface{}) ([]byte, error) {
			return msgpack.Marshal(v)
		},
		Unmarshal: func(b []byte, v interface{}) error {
			return msgpack.Unmarshal(b, v)
		},
	}
}

// expiration - expiration calculation
func (item *Item) expiration() time.Duration {
	if item.Expiration < 0 {
		return 0
	}
	if item.Expiration < time.Second {
		return time.Hour
	}
	return item.Expiration
}

// object - abstract caching object
func (item *Item) object() (interface{}, error) {
	if item.Object != nil {
		return item.Object, nil
	}
	return nil, nil
}

// setItem - write caching object
func (c *TCache) setItem(item *Item) ([]byte, error) {
	object, err := item.object()
	if err != nil {
		return nil, err
	}
	b, err := c.Marshal(object)
	if err != nil {
		log.Error().Msgf("cache: Marshal key=%q failed: %s", item.Key, err)
		return nil, err
	}
	if c.redis == nil {
		return b, nil
	}
	err = c.redis.Set(item.Key, b, item.expiration()).Err()
	if err != nil {
		log.Error().Msgf("cache: Set key=%q failed: %s", item.Key, err)
	}
	return b, err
}

// Exists - checking for an object in the cache by key
func (c *TCache) Exists(key string) bool {
	return c.Get(key, nil) == nil
}

// Get - getting object from cache by key
func (c *TCache) Get(key string, object interface{}) error {
	b, err := c.getBytes(key)
	if err != nil {
		return err
	}
	if object == nil || len(b) == 0 {
		return nil
	}
	err = c.Unmarshal(b, object)
	if err != nil {
		log.Error().Msgf("cache: key=%q Unmarshal(%T) failed: %s", key, object, err)
		return err
	}
	return nil
}

// getBytes - getting object in byte representation
func (c *TCache) getBytes(key string) ([]byte, error) {
	if c.redis == nil {
		return nil, ErrKeyCacheNotFound
	}
	b, err := c.redis.Get(key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrKeyCacheNotFound
		}
		log.Error().Msgf("cache: Get key=%q failed: %s", key, err)
		return nil, err
	}
	return b, nil
}

// Cache - main caching method
func (c *TCache) Cache(object interface{}, key string, ttl time.Duration, tags []string, data func() error) error {
	err := c.Get(key, object)
	if err == nil {
		return nil
	}
	err = data()
	if err != nil {
		return err
	}
	_, err = c.setItem(&Item{
		Key:        key,
		Object:     object,
		Expiration: ttl,
	})
	c.SetTags(key, tags)
	return nil
}

// getTagName - getting concatenated tag name
func (c *TCache) getTagName(tag string) string {
	return tagPrefixCache + tag
}

// SetTags - tagging
func (c *TCache) SetTags(key string, tags []string) {
	for _, tag := range tags {
		c.redis.SAdd(c.getTagName(tag), key)
	}
}

// Flush - deleting an object by key
func (c *TCache) Flush(key string) {
	c.redis.Del(key)
}

// FlushTags - remove objects by tag
func (c *TCache) FlushTags(tags []string) {
	for _, tag := range tags {
		keys := c.redis.SMembers(c.getTagName(tag))
		if len(keys.Val()) > 0 {
			keysWithTag := append(keys.Val(), tag)
			c.redis.Del(keysWithTag...)
		}
	}
}
