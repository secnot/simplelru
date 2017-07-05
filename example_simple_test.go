package simplelru_test

import (
	"fmt"
	"github.com/secnot/simplelru"
)

// Simple LRUCache example.
func ExampleNewLRUCache() {

	maxCacheSize := 1000
	pruneSize := 10

	cache := simplelru.NewLRUCache(maxCacheSize, pruneSize)

	// Fill the cache
	for i := 0; i < maxCacheSize; i++ {
		cache.Set(i, 2000+i)
	}
	fmt.Println(cache.Len()) //1000

	// Adding another item to the cache will trigger a prune
	cache.Set(3000, 5000)
	fmt.Println(cache.Len()) //991

	// Now the 10 oldest items aren't cached.
	if _, ok := cache.Get(0); !ok {
		fmt.Println("0 is not cached")
	}

	// To refresh an item just access it
	if _, ok := cache.Get(10); ok {
		fmt.Println("Now 10 is the most recent item")
	}

	// Now 11 is oldest the key
	cache.RemoveOldest()
	if _, ok := cache.Get(11); !ok {
		fmt.Println("11 was removed from cache")
	}
	if _, ok := cache.Get(10); ok {
		fmt.Println("10 is still here")
	}

	// Don't like the look of 29 either
	cache.Remove(29)

	// Output:
	// 1000
	// 991
	// 0 is not cached
	// Now 10 is the most recent item
	// 11 was removed from cache
	// 10 is still here
}
