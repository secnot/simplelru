package simplelru

import (
	"fmt"
	"testing"
	"time"
	"sync"
)


// Mock key:value storage for cache lookups (concurrency-safe)
////////////////////////////////////////////////////////////
type storage struct {
	storage map[interface{}]interface{}
	LookupCount int
	lock sync.Mutex
}

func newStorage(size int)(*storage) {

	s := storage{
		storage: make(map[interface{}]interface{}),
		LookupCount: 0,
		lock: sync.Mutex{},
	}

	for i := 0; i<size; i++ {
		s.storage[i] = i
	}

	return &s
}

func (s *storage)Get(key interface{})(value interface{}, ok bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.LookupCount++
	value, ok = s.storage[key]
	return
}
func (s *storage)CallCount() int{
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.LookupCount
}
func (s *storage)ResetCallCount(){
	s.lock.Lock()
	defer s.lock.Unlock()
	s.LookupCount = 0
	return
}
//////////////////////////////////////////////////////////////////




// Test basic lookup functionality (No concurrency or parallelism)
func TestBasicLookup(t *testing.T) {
	storage := newStorage(1000)

	lookup := func (key interface{}) (value interface{}, ok bool){
		time.Sleep(100 * time.Millisecond)
		return storage.Get(key)
	}

	cache := NewLookupLRUCache(100, 10, lookup, 1, 1000)
	
	// The key is not in cache so it should generage a lookup
	value, ok := cache.Get(77)
	if storage.CallCount() != 1 {
		t.Error("Lookup function was never called")
	}
	if !ok || value != 77  {
		t.Error(fmt.Sprintf("Expected 77, received %v", value))
	}
	hit, miss := cache.Stats()
	if hit!=0 || miss!=1 {
		t.Error("Stat accounting error")
	}

	// Now a query for the same key should be cached
	storage.ResetCallCount()
	value, ok = cache.Get(77)
	if storage.CallCount() != 0 {
		t.Error("There was a lookup for a key that should be cached")
	}
	if !ok || value != 77  {
		t.Error(fmt.Printf("Expected 77, received %v", value))
	}
	hit, miss = cache.Stats()
	if hit!=1 || miss!=1 {
		t.Error("Stat accounting error")
	}

	// Setting a key value overrides the lookup value
	cache.Set(88, 8888)
	value, ok = cache.Get(88)
	if !ok || value!=8888 {
		t.Error("The lookup should have been successful")
	}
	if storage.CallCount() != 0 {
		t.Error("There was a lookup for a key that should be cached")
	}	
	hit, miss = cache.Stats()
	if hit!=2 || miss!=1 {
		t.Error("Stat accounting error")
	}

	cache.Set(77, 11111)
	value, ok = cache.Get(77)
	if !ok || value!=11111 {
		t.Error("The lookup should have been successful")
	}
	if storage.CallCount() != 0 {
		t.Error("There was a lookup for a key that should be cached")
	}	
	hit, miss = cache.Stats()
	if hit!=3 || miss!=1 {
		t.Error("Stat accounting error")
	}

	// Request key not in cache or storage (return nil, false)
	initial_len := cache.Len()
	for i:=0; i<10; i++ {
		value, ok = cache.Get(1000)
		if value != nil || ok {
			t.Error(fmt.Sprintf("Should have returned nil, true not %v, %v", value, ok))
		}
		if initial_len != cache.Len() {
			t.Error("A failed Get shouldn't add anything to the DB")
		}
	}

	cache.Close()
}


// Test concurrent Get calls blocking into the same request generate only one lookup
func TestConcurrentGetRequests(t *testing.T) {
	storage := newStorage(1000)

	lookup := func (key interface{}) (value interface{}, ok bool){
		time.Sleep(100 * time.Millisecond)
		return storage.Get(key)
	}

	cache := NewLookupLRUCache(100, 10, lookup, 1, 1000)

	// Concurrent requests 
	concurrentGet := func(cache *LRUCache, key interface{}) {
		cache.Get(key)
	}
	go concurrentGet(cache, 100)
	go concurrentGet(cache, 100)

	if value, ok := cache.Get(100); !ok || value != 100 {
		t.Error("Get returned the wrong value")
	}
	if storage.CallCount() != 1 {
		t.Error("Concurrent Get calls should do a single lookup")
	}

	// Test lookups are sequential with a single go routine
	storage.ResetCallCount()
	go concurrentGet(cache, 40)
	go concurrentGet(cache, 50)
	go concurrentGet(cache, 40)
	go concurrentGet(cache, 50)
	go concurrentGet(cache, 60)

	value40, ok40 := cache.Get(40)
	value50, ok50 := cache.Get(50)
	value60, ok60 := cache.Get(60)

	if !ok40 || !ok50 || !ok60 {
		t.Error("Get request error")
	}
	if value40 != 40 || value50 != 50 || value60 != 60 {
		t.Error("Wrong key values")
	}

	if storage.CallCount() != 3 {
		t.Error(fmt.Sprintf("Used more lookups than expected"))
	}

	cache.Close()
}


// Test interrupting a Get lookup by th
func TestConcurrentGetSet(t *testing.T) {
	storage := newStorage(1000)

	lookup := func (key interface{}) (value interface{}, ok bool){
		time.Sleep(150 * time.Millisecond)
		return storage.Get(key)
	}

	cache := NewLookupLRUCache(100, 10, lookup, 1, 1000)

	// Concurrent requests 
	concurrentGet := func(cache *LRUCache, key interface{}, expected_value interface{}) {
		if value, ok := cache.Get(key); !ok || value != expected_value {
			t.Error("Get didn't receive the expected value")
		}
	}
	concurrentSet := func(cache *LRUCache, key interface{}, value interface{}) {
		cache.Set(key, value)
	}

	for i := 0; i < 10; i++ {
		go concurrentGet(cache, i, 3000)
		go concurrentGet(cache, i, 3000)
		time.Sleep(20*time.Millisecond)
		go concurrentSet(cache, i, 3000)
		time.Sleep(400*time.Millisecond)
	}

	for i := 0; i < 10; i++ {
		if value, ok := cache.Get(i); !ok || value != 3000 {
			t.Error(fmt.Sprintf("Get expected 3000, reveiced %v", value))
		}
	}
}
	
// Test with parallel lookup goroutines
func TestParallelLookupRequests(t *testing.T) {
	storage := newStorage(1000)

	lookup := func (key interface{}) (value interface{}, ok bool){
		// Sleep between 30-49ms
		time.Sleep(time.Duration((key.(int)%20)+30) * time.Millisecond)
		value, ok = storage.Get(key)
		time.Sleep(100 * time.Millisecond)
		return
	}

	cache := NewLookupLRUCache(100, 10, lookup, 500, 5000)

	// Concurrent requests 
	concurrentGet := func(cache *LRUCache, key interface{}) {
		cache.Get(key)
	}

	// 1500 concurrent Get requests without parallel lookups this is too slow
	for i:=0; i<500; i++ {
		go concurrentGet(cache, i)
		go concurrentGet(cache, i)
		go concurrentGet(cache, i)
	}

	// Wait enough time for all request to finish
	time.Sleep(1000*time.Millisecond)

	// Get a new value to assure everithing is finished
	if value, ok := cache.Get(501); !ok || value != 501 {
		t.Error("Get returned the wrong value")
	}
	if storage.CallCount() != 501 {
		t.Error("Concurrent Get calls should do a single lookup")
	}

	// Test lookups are sequential with a single go routine
	storage.ResetCallCount()
	go concurrentGet(cache, 540)
	go concurrentGet(cache, 550)
	go concurrentGet(cache, 540)
	go concurrentGet(cache, 550)
	go concurrentGet(cache, 560)

	value40, ok40 := cache.Get(540)
	value50, ok50 := cache.Get(550)
	value60, ok60 := cache.Get(560)

	if !ok40 || !ok50 || !ok60 {
		t.Error("Get request error")
	}
	if value40 != 540 || value50 != 550 || value60 != 560 {
		t.Error("Wrong key values")
	}

	if storage.CallCount() != 3 {
		t.Error(storage.CallCount())
		t.Error(fmt.Sprintf("Used more lookups than expected"))
	}

	cache.Close()
}


// Basic Set tests
func TestLookupSet(t *testing.T) {
	storage := newStorage(1000)

	// lookup func has 500ms delay
	lookup := func (key interface{}) (value interface{}, ok bool){
		time.Sleep(400 * time.Millisecond)
		value, ok = storage.Get(key)
		return
	}
	cache := NewLookupLRUCache(10000, 100, lookup, 5, 1000)

	// Lookup some initial values
	if value, ok := cache.Get(10); !ok || value != 10 {
		t.Error("Get: Lookup Error")
	}
	cache.Get(100)
	cache.Get(10000)

	// Set value with a 0-9ms delay
	concurrentSet := func (cache *LRUCache, key interface{}, value interface{}) {
		time.Sleep(time.Duration(key.(int)%10)* time.Millisecond)
		cache.Set(key, value)
	}
	
	for i := 0; i < 5000; i++ {
		go concurrentSet(cache, i, i+9000)
	}

	// Wait for all Set calls to finish
	time.Sleep(1000*time.Millisecond)

	// Verify all values were Set
	for i := 0; i < 5000; i++ {
		if value, ok := cache.Get(i); !ok || value.(int) != i+9000 {
			t.Error("There was an error while setting cache values")
		}
	}

	cache.Close()	
}


// Test interrupting lookup operations by Setting the key value
func TestLookupInterrupt(t *testing.T) {

	storage := newStorage(1000)

	// lookup func has 500ms delay
	lookup := func (key interface{}) (value interface{}, ok bool){
		time.Sleep(400 * time.Millisecond)
		value, ok = storage.Get(key)
		return
	}

	cache := NewLookupLRUCache(100, 10, lookup, 10, 1000)
	
	//
	concurrentGet := func(cache *LRUCache, key interface{}) {
		cache.Get(key)
	}

	for i := 0 ; i < 5; i++ {
		// Launch lookup
		go concurrentGet(cache, 100)

		// Set the cache value to something different from the storage value
		cache.Set(100, 12345)

		// Verify the cache has stored the new value
		if value, ok := cache.Get(100); !ok || value != 12345 {
			t.Error("Set didn't change the value returned by the lookup")
		}

		// The lookup shouldn't have finished yet
		if count := storage.CallCount(); count != 0 {
			t.Error("Ops the lookup was too fast for the test to be thorough")
		}

		// Wait a few seconds to assure the lookup has had time to finish
		time.Sleep(800*time.Millisecond)
		
		if value, ok := cache.Get(100); !ok || value!=12345 {
			t.Error("lookup function modified the value updated by Set")
		}
	}
	
	cache.Close()
}


// Test peek with lookup enabled
func TestLookupPeek(t *testing.T) {

	storage := newStorage(1000)

	// lookup func has 500ms delay
	lookup := func (key interface{}) (value interface{}, ok bool){
		value, ok = storage.Get(key)
		return
	}

	cache := NewLookupLRUCache(100, 10, lookup, 10, 1000)


	// Peek unknown key
	if _, ok := cache.Peek(100); ok {
		t.Error("Peek shouldn't have generated a lookup")
	}

	time.Sleep(100*time.Millisecond) // Wait in case there was a lookup
	
	if storage.CallCount() != 0 {
		t.Error("Peek shouldn't have generated a lookup")
	}

	if hit, miss := cache.Stats(); hit != 0 || miss != 0 {
		t.Error("Peek shouldn't update the stats")
	}

	// Peek existing key
	cache.Set(100, 1000)
	
	if value, ok := cache.Peek(100); !ok || value != 1000 {
		t.Error("Peek didn't return the cached value")
	}
	if storage.CallCount() != 0 {
		t.Error("Peek shouldn't have generated a lookup")
	}
	if hit, miss := cache.Stats(); hit != 0 || miss != 0 {
		t.Error("Peek shouldn't update the stats")
	}
}	
	

// Operate with two caches to verify there is no shared state
func TestLookupDualCaches(t *testing.T) {
	storage := newStorage(1000)

	// lookup func has random 0-9ms delay
	lookup := func (key interface{}) (value interface{}, ok bool){
		time.Sleep(time.Duration(key.(int)%10)* time.Millisecond)
		value, ok = storage.Get(key)
		return
	}

	concurrentSet := func (cache *LRUCache, key interface{}, value interface{}) {
		time.Sleep(time.Duration(key.(int)%10)* time.Millisecond)
		cache.Set(key, value)
	}

	// Queue size 10
	cache1 := NewLookupLRUCache(1000, 100, lookup, 8, 10)
	cache2 := NewLookupLRUCache(1000, 100, lookup, 8, 10)

	// Set different values for botch caches)
	for i := 0; i < 500; i++{
		go concurrentSet(cache1, i, i+1000)
		go concurrentSet(cache1, i, i+1000)
		go concurrentSet(cache2, i, i+2000)
		go concurrentSet(cache2, i, i+2000)
	}

	// Wait until all requests are finished
	time.Sleep(2000*time.Millisecond)

	// Verify results
	for i := 0; i < 500; i++ {
		if value, ok := cache1.Get(i); !ok || value != i+1000 {
			t.Error("cache1 was not updated successfully")
		}
		if value, ok := cache2.Get(i); !ok || value != i+2000 {
			t.Error("cache2 was not updated successfully")
		}
	}
	for i :=500; i < 1000; i++ {
		if value, ok := cache1.Get(i); !ok || value != i {
			t.Error("cache1 was not updated successfully")
		}
		if value, ok := cache2.Get(i); !ok || value != i {
			t.Error("cache2 was not updated successfully")
		}
	}

}


// Test full lookup request queue
func TestLookupFullChannel(t *testing.T) {

	storage := newStorage(1000)

	// lookup func has random 0-9ms delay
	lookup := func (key interface{}) (value interface{}, ok bool){
		time.Sleep(time.Duration(key.(int)%10)* time.Millisecond)
		value, ok = storage.Get(key)
		return
	}

	// Queue size 10
	cache := NewLookupLRUCache(100, 10, lookup, 8, 10)
	
	//
	concurrentGet := func(cache *LRUCache, key interface{}) {
		cache.Get(key)
	}	
	concurrentSet := func(cache *LRUCache, key interface{}, value interface{}) {
		cache.Set(key, value)
	}	
	
	
	for i:=0; i<100; i++ {
		go concurrentGet(cache, i)
		go concurrentSet(cache, i+2000, i+2000)
		go concurrentGet(cache, i)
	}

	// wait enough time for all the lookups to finish
	time.Sleep(4000*time.Millisecond)

	cache.Close()
}
