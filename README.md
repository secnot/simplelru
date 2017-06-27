# simplelru
LRU Cache implementation written Go



## Usage

Basic LRU cache example

```go
package main

import (
	"fmt"
	"github.com/secnot/simplelru"
)

func main() {

	maxCacheSize := 1000
	pruneSize := 10

	cache := simplelru.NewLRUCache(maxCacheSize, pruneSize)

	// Add or update cache items with Set
	cache.Set("John Smith", 32)
	cache.Set("John Smith", 33)

	// Use Get to retrieve item value
	if value, ok := cache.Get("Little Pony") {
		fmt.Println("Little Pony isn't cached")
	}

	// How many items are in the cache
	fmt.Printf("There are %v cached items\n", cache.Len())

	// John Smith looks like a fake name better remove it
	cache.Remove("John Smith")
}
```

Or on how to use a fetch function on cache misses:

```go
package main

import (
	"fmt"
	"time"

	"github.com/secnot/simplelru"
)

const (
	maxCacheSize = 10000
	pruneSize = 100
	fetchWorkers = 2
	fetchQueueSize = fetchWorkers * 2
)


// Simulated db query, with delay
func mockDBFetch(key interface{}) (value interface{}, ok bool) {
	time.Sleep(20 * time.Millisecond)
	return fmt.Sprintf("query result: %v", key), true
}


func main() {

	cache := simplelru.NewFetchingLRUCache(
		maxCacheSize,
		pruneSize,
		mockDBFetch,
		fetchWorkers,
		fetchQueueSize)

	// If a key is not cached there is a fetch
	value, _ := cache.Get("John")
	fmt.Println(value) // query result: John

	// No fetch for cached keys
	cache.Set("Mary", "Mary is cached")
	value, _ = cache.Get("Mary")
	fmt.Println(value) // Mary is cached
}
``` 


## TODO: 
