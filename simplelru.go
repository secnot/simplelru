package simplelru

import (
	"github.com/secnot/orderedmap"
	"sync"
	"fmt"
)

// LookupFunc is used to loook up missing values when there is a miss
type FetchFunc  func (key interface{}) (value interface{}, ok bool)



type fetchRequest struct {
	value interface {}
	ok bool
	ready chan struct{} //Close when request is ready
}

func newFetchRequest() *fetchRequest{
	return &fetchRequest{
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

	// Elements pruned everytime the cache if full
	pruneSize int

	// Hit miss stats
	hitCount  uint64
	missCount uint64

	// Lookup function for missing keys
	fetcher FetchFunc

	// Map and queue of keys waiting to be fetched
	fetchM map[interface{}]*fetchRequest
	fetchQ chan interface{} // lookup request key queue
}


// Value fetching worker goroutine
func (c *LRUCache) goFetchWorkerFunc() {

	defer c.wg.Done()	
	for {
		// Next key for lookup
		key, ok := <-c.fetchQ
		if !ok {
			return // Received exit signal
		}

		// Check the request for the keys is still waiting and hasn't been 
		// removed by a Set call
		c.Lock()
		if _, ok := c.fetchM[key]; !ok {
			c.Unlock()
			continue
		}
		c.Unlock()

		// Use fetch function
		value, fetchOk := c.fetcher(key)
		if !fetchOk {
			// If the lookup failed discard the value as a precaution
			value = nil
		}

		// Check once more if the request was removed from fetchM,
		// if not, set the value and signal waiting goroutines
		c.Lock()
		if request, stillWaiting := c.fetchM[key]; stillWaiting { 	
			request.value = value
			request.ok = fetchOk

			// All blocked Get methods keep a reference, so it can
			// be deleted safely
			delete(c.fetchM, key)

			// Clossing the channel marks the request finished
			close(request.ready)

			// Only update the cache if fetching was successful
			if fetchOk {
				c.cache.Set(key, value)
			}
		} 
		c.Unlock()
	}
}


// New LRUCache with fetch function to retrieve keys on cache misses.
// 
// If fetchWorkers is greater than one, fetch function must be 
// concurrency-safe.
//
// fetchQueueSize must be selected depending on the number of workers and 
// expected concurrent cache misses.
func NewFetchingLRUCache(size int, pruneSize int, 
					   fetcher FetchFunc, 
					   fetchWorkers uint32,  
					   fetchQueueSize uint32) *LRUCache {
	if size < 1 {
		panic("NewFetchingLRUCache: min cache size is 1")
	}
	if pruneSize < 1 {
		panic("NewFetchingLRUCache: min prune size is 1")
	}
	if fetcher != nil && fetchWorkers < 1 {
		panic("NewFetchingLRUCache: The min worker pool size is 1")
	}
	if fetcher != nil && fetchQueueSize < 1{
		panic("NewFetchingLRUCache: The min fetch job queue size is 1")
	}

	cache := &LRUCache {
		cache: orderedmap.NewOrderedMap(),
		size: size, 
		pruneSize: pruneSize,
		hitCount: 0,
		missCount: 0,
		fetcher: fetcher,
		fetchM: make(map[interface{}]*fetchRequest),
		fetchQ: make(chan interface{}, fetchQueueSize),
	}

	if fetcher != nil {
		for i := uint32(0); i < fetchWorkers; i++ {
			cache.wg.Add(1)
			go cache.goFetchWorkerFunc()
		}
	}

	return cache

}


// New LRUCache without lookup function
func NewLRUCache(size int, pruneSize int) *LRUCache {
	return NewFetchingLRUCache(size, pruneSize, nil, 0, 0)
}


// Remove pruneSize elements from cache
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


// Get the key value, if not cached use the fetch function if available.
func (c *LRUCache) Get(key interface{}) (value interface{}, ok bool){
	c.Lock()
	
	if value, ok = c.cache.Get(key); ok {
		c.hitCount++
		c.cache.MoveLast(key)
		c.Unlock()
	} else if c.fetcher != nil {
		c.missCount++
		request, exists := c.fetchM[key]
		if !exists { // Start new request
			request = newFetchRequest()
			c.fetchM[key] = request
			c.Unlock()
			c.fetchQ <- key // Queue key for fetch
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


// Set or update key value, returns true if the cache was pruned to make space
// for a new key. Set has priority over fetched values, so if the key set is
// being fetched, all goroutines waiting will wakeup and receive the 'setted' value
// while the fetch results are discarded.
func (c *LRUCache) Set(key interface{}, value interface{}) (pruned bool){
	c.Lock()

	inCache := false

	if _, inCache = c.cache.Get(key); inCache { 
		// Already in cache, just update
		c.cache.MoveLast(key)
	} else if request, fetching := c.fetchM[key]; fetching {
		// In lookup queue (but not in cache)
		request.value = value
		request.ok = true	
		
		// All blocked Get methods keep a reference so it can be deleted safely
		delete(c.fetchM, key)

		// Clossing the channel marks request finished
		close(request.ready)
	}
	
	if !inCache && c.cache.Len() >= c.size {
		c.prune()
		pruned = true
	} else {
		pruned = false
	}

	// The new value is set after the purge to assure it is not deleted 
	// when the cache size is one, or the prune size is greater than cache size
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


// Remove oldest/least used key from cache
func (c *LRUCache) RemoveOldest() {
	c.Lock()
	c.cache.PopFirst()
	c.Unlock()
}


// Get key value without updating cache, stats, or triggering a fetch
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


// Stop all fetch routines
func (c *LRUCache) Close() {
	c.Lock()
	close(c.fetchQ)
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
