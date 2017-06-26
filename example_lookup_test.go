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

	// Workers in the lookup pool
	lookupWorkers = 2

	// Lookup workers request queue size for worker
	lookupQueueSize = lookupWorkers * 2
)


// Simulated db lookup, if there is more than one worker in the pool the
// function must be concurrency-safe. (type: LookupFunc)
func mockDBLookup(key interface{}) (value interface{}, ok bool) {
	// Simulate lookup delay
	time.Sleep(20 * time.Millisecond)

	// If ok is true the lookup was successful, and the return value
	// is returned and stored into the cache.
	return fmt.Sprintf("query result: %v", key), true
}


// ExampleNewLookupLRUCache demostrates how to create a LRUCache with lookup functionality.
func Example_NewLookupLRUCache() {

	cache := simplelru.NewLookupLRUCache(
		maxCacheSize,
		pruneSize,
		mockDBLookup,
		lookupWorkers,
		lookupQueueSize)

	// The key is not cached so there is a lookup
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
