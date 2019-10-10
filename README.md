# redis-tcache
Tagging radis caching library (based in go-redis)

## Example №1 (wrapper cache): 

```golang
var cacheModel Model
  
err := tcache.Cache(&cacheModel, "unique-cache-key", 6 * time.Hour, []string{"tag1", "tag2", "tag3"}, func() error {
   cacheModel, err := repo.FillCahceModelFromDB()
   if err != nil {
     return err
   }
   return nil
})  
```

## Example №2
```golang
if err := tcache.Get("unique-cache-key", &cacheModel); err != nil {
    cacheModel, err := repo.FillCahceModelFromDB()
    if err != nil {
       return err
    }
    _, err = tcache.Set(&tcache.Item{
      Key:        "unique-cache-key",
      Object:     cacheModel,
      Expiration: 6 * time.Hour,
    })
    tcache.SetTags("unique-cache-key", []string{"tag1", "tag2", "tag3"})
}
```

## Flush cache

```golang
// flush by tags
tcache.FlushTags([]string{"tag1", "tag2"})
// flush by key
tcache.Flush(""unique-cache-key")

```
## Сustomize serialization (Example: protobuf JSONPb)
```golang
func R() *tcache.TCache {
   c := tcache.NewTCache(Client)

   c.Marshal = func(i interface{}) (b []byte, e error) {
	customMarshaller := &runtime.JSONPb{
		OrigName:     true,
		EmitDefaults: true, // disable 'omitempty'
	}
	return customMarshaller.Marshal(i)
    }
    c.Unmarshal = func(bytes []byte, i interface{}) error {
	customMarshaller := &runtime.JSONPb{
		OrigName:     true,
		EmitDefaults: true, // disable 'omitempty'
	}
	return customMarshaller.Unmarshal(bytes, i)
     }

   return c
}
```
