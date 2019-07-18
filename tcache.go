package tcache

import (
	"errors"
	"sync/atomic"
	"time"

	"github.com/vmihailenco/msgpack"

	"github.com/go-redis/redis"
	"github.com/rs/zerolog/log"
)

var ErrCacheMiss = errors.New("cache: key is missing")

const (
	tagPrefixCache = "tag:"
)

type Item struct {
	Key        string
	Object     interface{}
	Expiration time.Duration
}

type TCache struct {
	redis     *redis.Client
	Marshal   func(interface{}) ([]byte, error)
	Unmarshal func([]byte, interface{}) error
	misses    uint64
	hits      uint64
}

func (item *Item) exp() time.Duration {
	if item.Expiration < 0 {
		return 0
	}
	if item.Expiration < time.Second {
		return time.Hour
	}
	return item.Expiration
}

func (item *Item) object() (interface{}, error) {
	if item.Object != nil {
		return item.Object, nil
	}
	return nil, nil
}

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
	err = c.redis.Set(item.Key, b, item.exp()).Err()
	if err != nil {
		log.Error().Msgf("cache: Set key=%q failed: %s", item.Key, err)
	}
	return b, err
}

func (c *TCache) Exists(key string) bool {
	return c.Get(key, nil) == nil
}

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

func (c *TCache) getBytes(key string) ([]byte, error) {
	if c.redis == nil {
		return nil, ErrCacheMiss
	}
	b, err := c.redis.Get(key).Bytes()
	if err != nil {
		atomic.AddUint64(&c.misses, 1)
		if err == redis.Nil {
			return nil, ErrCacheMiss
		}
		log.Error().Msgf("cache: Get key=%q failed: %s", key, err)
		return nil, err
	}
	atomic.AddUint64(&c.hits, 1)
	return b, nil
}

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

func (c *TCache) getTagName(tag string) string {
	return tagPrefixCache + tag
}

func (c *TCache) SetTags(key string, tags []string) {
	for _, tag := range tags {
		c.redis.SAdd(c.getTagName(tag), key)
	}
}

func (c *TCache) Flush(key string) {
	c.redis.Del(key)
}

func (c *TCache) FlushTags(tags []string) {
	for _, tag := range tags {
		keys := c.redis.SMembers(c.getTagName(tag))
		if len(keys.Val()) > 0 {
			keysWithTag := append(keys.Val(), tag)
			c.redis.Del(keysWithTag...)
		}
	}
}
