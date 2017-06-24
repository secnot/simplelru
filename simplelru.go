package simplelru

import (
	"github.com/secnot/orderedmap"
	"sync"
	"fmt"
)

// LookupFunc is used to loook up missing values when there is a miss
type LookupFunc  func (key interface{}) (value interface{}, ok bool)




type lookupRequest struct {
	value interface {}
	ok bool
	ready chan struct{} //Close when request is ready
}

func newLookupRequest() *lookupRequest{
	return &lookupRequest{
		value: nil,
		ok: false,
		ready: make(chan struct{}),
	}
}


type LRUCache struct{

	// Wait for lookup task exits
	wg sync.WaitGroup

	// Embedded mutex
	sync.Mutex
	
	// 
	cache *orderedmap.OrderedMap

	// Max Size
	size int
	pruneSize int

	// Hit miss stats
	hitCount  uint64
	missCount uint64

	// Lookup function for missing keys
	lookup LookupFunc

	// Keys waiting for lookup
	lookupM map[interface{}]*lookupRequest
	lookupQ chan interface{} // lookup request key queue
}


func goRequestLookupFunc(c *LRUCache) {

	//var key interface{}
	defer c.wg.Done()	
	for {
		// Next key for lookup
		key, ok := <-c.lookupQ
		if !ok {
			return // Received exit signal
		}

		// Check the key is still in lookupM and wasn't removed by a Set call
		c.Lock()
		if _, ok := c.lookupM[key]; !ok {
			c.Unlock()
			continue
		}
		c.Unlock()


		// Call lookup function and return value
		value, lookupOk := c.lookup(key)
		
		c.Lock()
		// If the key was removed from the lookupMap it means a Set call updated
		// the value, and it has precedence over this
		if request, stillWaiting := c.lookupM[key]; stillWaiting { 	
			request.value = value
			request.ok = lookupOk

			// All blocked Get methods should keep a reference
			delete(c.lookupM, key)

			// Clossing the channel marks request finished
			close(request.ready)

			// Only update the cache if the lookup was successful
			if lookupOk {
				c.cache.Set(key, value)
			}
		} 
		c.Unlock()
	}
}


// New LRUCache with lookup
func NewLookupLRUCache(size int, pruneSize int, 
					   lookup LookupFunc, 
					   lookupPoolSize uint16,  
					   lookupQueueSize uint32) *LRUCache {
	if size < 1 {
		panic("NewLookupLRUCache: min size is 1")
	}
	if pruneSize < 1 {
		panic("NewLookupLRUCache: min pruneSize is 1")
	}
	if lookup != nil && lookupPoolSize == 0 {
		panic("NewLookupLRUCache: If a lookup function is provided the min pool size is 1")
	}
	if lookup != nil && lookupQueueSize < 1{
		panic("NewLookupLRUCache: If a lookup function is provided the min queue size is 1")
	}

	cache := &LRUCache {
		cache: orderedmap.NewOrderedMap(),
		size: size, 
		pruneSize: pruneSize,
		hitCount: 0,
		missCount: 0,
		lookup: lookup,
		lookupM: make(map[interface{}]*lookupRequest),
		lookupQ: make(chan interface{}, lookupQueueSize),
	}

	for i := uint16(0); i < lookupPoolSize; i++ {
		cache.wg.Add(1)
		go goRequestLookupFunc(cache)
	}

	return cache

}


// New LRUCache without lookup function
func NewLRUCache(size int, pruneSize int) *LRUCache {
	return NewLookupLRUCache(size, pruneSize, nil, 0, 0)
}


// Remove prune_size elements from cache
func (c *LRUCache) prune() {
	for x:=c.pruneSize; x>0; x-- {
		if _, _, ok := c.cache.PopFirst(); !ok {
			break // Cache is already empty
		}
	}
}


// Return number of keys in cache
func (c *LRUCache) Len() (size int){
	c.Lock()
	size = c.cache.Len()
	c.Unlock()
	return
}


// Get key value, if it is not available returns nil
func (c *LRUCache) Get(key interface{}) (value interface{}, ok bool){
	c.Lock()
	
	if value, ok = c.cache.Get(key); ok {
		c.hitCount++
		c.cache.MoveLast(key)
		c.Unlock()
	} else if c.lookup != nil {
		c.missCount++
		request, exists := c.lookupM[key]
		if !exists { // Start new request
			request = newLookupRequest()
			c.lookupM[key] = request
			c.Unlock()
			c.lookupQ <- key // Queue key for lookup
		} else {
			c.Unlock()
		}
		
		// Wait until the lookup has finished
		<-request.ready // Wait until lookup is done
		value, ok = request.value, request.ok
	} else {
		c.missCount++
		c.Unlock()
	}
	return
}


// Set or update key value, return true the cache was full and some other key
// was purged to make space.
func (c *LRUCache) Set(key interface{}, value interface{}) (purged bool){
	c.Lock()

	inCache := false
	purged = false

	if _, inCache = c.cache.Get(key); inCache { 
		// Already in cache, just update
		c.cache.MoveLast(key)
	} else if request, inLookup := c.lookupM[key]; inLookup {
		// In lookup queue (but not in cache)
		request.value = value
		request.ok = true	
		
		// All blocked Get methods should keep a reference
		delete(c.lookupM, key)

		// Clossing the channel marks request finished
		close(request.ready)
	}
	
	if !inCache && c.cache.Len() == c.size {
		c.prune()
		purged = true
	}

	// The new value is set after the purge to assure it is not deleted 
	// in very small caches
	c.cache.Set(key, value)
	c.Unlock()
	return
}


// Remove key from cache
func (c *LRUCache) Remove(key interface{}) {
	c.Lock()
	c.cache.Delete(key)
	c.Unlock()
}


// Remove oldest key from cache
func (c *LRUCache) RemoveOldest() {
	c.Lock()
	c.cache.PopFirst()
	c.Unlock()
}


// Get key value without updating cache, stats, or triggering a lookup
func (c *LRUCache) Peek(key interface{}) (value interface{}, ok bool){
	c.Lock()
	value, ok = c.cache.Get(key)
	c.Unlock()
	return
}


// Returns true if the cache contains the key (no side-effects)
func (c *LRUCache) Contains(key interface{}) bool{
	_, ok := c.Peek(key)
	return ok
}


// Purge all cache contents (without reseting stats)
func (c *LRUCache) Purge() {
	c.Lock()
	c.cache = orderedmap.NewOrderedMap()
	c.Unlock()
}


// Stop all lookup routines
func (c *LRUCache) Close() {
	c.Lock()
	close(c.lookupQ)
	c.Unlock()
	c.wg.Wait()
}


// Return cache stats
func (c *LRUCache) Stats() (hit uint64, miss uint64) {
	c.Lock()
	hit, miss = c.hitCount, c.missCount
	c.Unlock()
	return
}


// Reset cache stats
func (c *LRUCache) ResetStats() {
	c.Lock()
	c.hitCount  = 0
	c.missCount = 0
	c.Unlock()
}


// Stringer interface
func (c *LRUCache) String() string {
	c.Lock()
	defer c.Unlock()
	return c.cache.String()
	return fmt.Sprintf("LRUCache(%v, %v)", c.size, c.cache.Len())
}
