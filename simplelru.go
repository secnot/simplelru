package simplelru

import (
	"github.com/secnot/orderedmap"
	"sync"
	"fmt"
)


// LookupFunc is used to loook up missing values when there is a miss
type LookupFunc  func (key interface{}) (value interface{}, ok bool)

// PruneCallback if called when a cache entry is pruned
type PruneCallback func (key interface{}, value interface{})

type LRUCache struct{
	cache *orderedmap.OrderedMap

	// Max Size
	size int
	prune_size int

	// Hit miss stats
	hit_count  uint64
	miss_count uint64

	//
	lock sync.Mutex

	// Lookup function for missing keys
	lookup LookupFunc

	// Called for each key that is pruned from cache
	onPrune PruneCallback
}



// New initialized LRUCache
func NewLRUCache(size int, prune_size int) *LRUCache {
	if size < 1 {
		panic("NewLRUCache: size out of range")
	}
	if prune_size < 1 {
		panic("NewLRUCache: prune_size out of range")
	}
	cache := &LRUCache {
		cache: orderedmap.NewOrderedMap(),
		size: size, 
		prune_size: prune_size,
		hit_count: 0,
		miss_count: 0,
		lookup: nil,
		onPrune: nil,
	}

	return cache
}


// Remove prune_size elements from cache
func (c *LRUCache) prune() {
	for x:=c.prune_size; x>0; x-- {
		if _, _, ok := c.cache.PopFirst(); !ok {
			break // Cache is already empty
		}
	}
}


// Return number of keys in cache
func (c *LRUCache) Len() (size int){
	c.lock.Lock()
	size = c.cache.Len()
	c.lock.Unlock()
	return
}


// Get key value, if it is not available returns nil
func (c *LRUCache) Get(key interface{}) (value interface{}, ok bool){
	c.lock.Lock()
	if value, ok = c.cache.Get(key); ok {
		c.hit_count++
		c.cache.MoveLast(key)
	} else {
		c.miss_count++
	}
	c.lock.Unlock()
	return
}


// Set or update key value, return true the cache was full and some other key
// was purged to make space.
func (c *LRUCache) Set(key interface{}, value interface{}) (purged bool){
	c.lock.Lock()

	purged = false
	if _, ok := c.cache.Get(key); ok { 
		// Already in cache
		c.cache.Set(key, value)
		c.cache.MoveLast(key)
	} else { 
		// New key
		if c.cache.Len() == c.size {
			c.prune()
			purged = true
		}
		c.cache.Set(key, value)
	}

	c.lock.Unlock()
	return
}


// Remove key from cache
func (c *LRUCache) Remove(key interface{}) {
	c.lock.Lock()
	c.cache.Delete(key)
	c.lock.Unlock()
}


// Remove oldest key from cache
func (c *LRUCache) RemoveOldest() {
	c.lock.Lock()
	c.cache.PopFirst()
	c.lock.Unlock()
}


// Get key value without updating cache or stats
func (c *LRUCache) Peek(key interface{}) (value interface{}, ok bool){
	c.lock.Lock()
	value, ok = c.cache.Get(key)
	c.lock.Unlock()
	return
}


// Returns true if the cache contains the key (no side-effects)
func (c *LRUCache) Contains(key interface{}) bool{
	_, ok := c.Peek(key)
	return ok
}


// Purge all cache contents (without reseting stats)
func (c *LRUCache) Purge() {
	c.lock.Lock()
	c.cache = orderedmap.NewOrderedMap()
	c.lock.Unlock()
}


// Return cache stats
func (c *LRUCache) Stats() (hit uint64, miss uint64) {
	c.lock.Lock()
	hit, miss = c.hit_count, c.miss_count
	c.lock.Unlock()
	return
}


// Reset cache stats
func (c *LRUCache) ResetStats() {
	c.lock.Lock()
	c.hit_count  = 0
	c.miss_count = 0
	c.lock.Unlock()
}


// Stringer interface
func (c *LRUCache) String() string {
	c.lock.Lock()
	defer c.lock.Unlock()
	return fmt.Sprintf("LRUCache(%v, %v)", c.size, c.cache.Len())
}
