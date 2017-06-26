package simplelru_test

import (
	"fmt"
	"time"

	"github.com/secnot/simplelru"
)

const (
	// Max cache elements before a prune
	maxCacheSize = 10000

	// Elements removed everytime there is prune
	pruneSize = 100

	// Workers in the fetch pool
	fetchWorkers = 2

	// fetch request jobs queue size
	// (Tune for each application)
	fetchQueueSize = fetchWorkers * 2
)


// Simulated db query, if there is more than one worker in the pool the
// function must be concurrency-safe. (type: FetchFunc)
func mockDBFetch(key interface{}) (value interface{}, ok bool) {
	// Simulate lookup delay
	time.Sleep(20 * time.Millisecond)

	// If ok is true the query was successful, and the return value
	// is returned and stored into the cache.
	return fmt.Sprintf("query result: %v", key), true
}


// Demostrates how to create a LRUCache with fetching functionality.
func ExampleNewFetchingLRUCache() {

	cache := simplelru.NewFetchingLRUCache(
		maxCacheSize,
		pruneSize,
		mockDBFetch,
		fetchWorkers,
		fetchQueueSize)

	// The key is not cached so fetch function is called
	value, _ := cache.Get("John")
	fmt.Println(value)

	// No lookup for cached keys
	cache.Set("Mary", "Not a lookup, key is cached")
	value, _ = cache.Get("Mary")
	fmt.Println(value)

	// Output:
	// query result: John
	// Not a lookup, key is cached
}
