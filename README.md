# redis-tcache
Tagging radis caching library (based in go-redis)

Example №1 (wrapper cache): 

```golang
  var cacheModel Model
  
	err := cache.Cache(&cacheModel, "unique-cache-key", 6 * time.Hour, []string{"tag1", "tag2", "tag3"}, func() error {
    cacheModel := repo.FillCahceModelFromDB()
		if err != nil {
			return err
		}
		return nil
	})
  
```
