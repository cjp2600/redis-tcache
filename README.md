# redis-tcache
Tagging radis caching library (based in go-redis)

## Example №1 (wrapper cache): 

```golang
var cacheModel Model
  
err := cache.Cache(&cacheModel, "unique-cache-key", 6 * time.Hour, []string{"tag1", "tag2", "tag3"}, func() error {
   cacheModel, err := repo.FillCahceModelFromDB()
   if err != nil {
     return err
   }
   return nil
})  
```

## Flush cache

```golang
  // flush by tags
	cache.FlushTags([]string{"tag1", "tag2"})
  // flush by key
	cache.Flush(""unique-cache-key")
```
