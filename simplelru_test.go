package simplelru

import (
	"fmt"
	"time"
	"testing"
	"strconv"
)



func TestNewLRUCache(t *testing.T) {

	cache := NewLRUCache(100, 10)

	// Test initialization
	if cache.size != 100 {
		t.Error("Unexpected max cache size")
	}

	if cache.pruneSize != 10 {
		t.Error("Unexpected prune_size")
	}

	if cache.hitCount != 0 {
		t.Error("hit_count initialization error")
	}

	if cache.missCount != 0 {
		t.Error("miss_count initialization error")
	}

	if cache.Len() != 0 {
		t.Error("the cache should be empty")
	}
}



func TestPurge(t *testing.T) {
	cache := NewLRUCache(100, 10)

	cache.Set("11", 11)
	cache.Set("12", 12)
	if cache.Len() != 2 {
		t.Error("Unexpected cache length")
	}

	if value, ok := cache.Get("11"); value!=11 || !ok {
		t.Error("11 Should have been cached")
	}
	if value, ok := cache.Get("12"); value!=12 || !ok {
		t.Error("12 Should have been cached")
	}
	
	cache.Purge()

	if cache.Len() != 0 {
		t.Error("Cache should have been empty")
	}

	if _, ok := cache.Get("11"); ok {
		t.Error("Cache should have been empty")
	}
	if _, ok := cache.Get("12"); ok {
		t.Error("Cache should have been empty")
	}

	cache.Close()
}


func TestSet(t *testing.T) {
	cache := NewLRUCache(100, 10)
	for i:=0; i<100; i++ {
		cache.Set(fmt.Sprintf("%v", i), i)
	}
	if cache.Len() != 100 {
		t.Error("The cache cache was pruned before time")
	}

	for i:=0; i<100; i++ {
		key := fmt.Sprintf("%v", i)
		if value, ok := cache.Get(key); value != i || !ok {
			t.Error(fmt.Sprintf("Expecting %v not %v", i, value))
		}
	}


	// Test cache pruning, adding one more key should prune 'prune_size'
	cache.Set("1000", 1000)
	if cache.Len() != 91 {
		t.Error("Pruning wasn't successful")
	}

	// Test 10 oldest keys were pruned
	for i:=0; i<10; i++ {
		key := fmt.Sprintf("%v", i)
		if _, ok := cache.Get(key); ok {
			t.Error(fmt.Sprintf("%v Should have been purged", key))
		}
	}

	if _, ok := cache.Get("11"); !ok {
		t.Error("'10' Should still be cached")
	}
	if _, ok := cache.Get("1000"); !ok {
		t.Error("Last key should still be cached")
	}

	// More pruning
	cache = NewLRUCache(1, 1000)
	cache.Set(1, 1)
	cache.Set(2, 2)
	if _, ok := cache.Get(2); !ok {
		t.Error("2 should be in the cache")
	}
	if cache.Len() != 1 {
		t.Error("Max size was ignored")
	}

	// Test it returns true when there is prunning
	cache = NewLRUCache(100, 10)
	for i:=0; i<100; i++ {
		if prune := cache.Set(fmt.Sprintf("%v", i), i); prune {
			t.Error("Set called prune when there is enough space in the cache")
		}
	}

	if prune := cache.Set(100000, 100000); !prune {
		t.Error("This should have called prune")
	}

	// Test updating a key doesn't prune the cache, only refreshes its age
	cache = NewLRUCache(100, 10)
	for i:=0; i<100; i++{
		cache.Set(i, i)
	}
	cache.Set(0, 500)

	if cache.Len() != 100 {
		t.Error(cache.Len())
		t.Error("Updating a keys shouldn't trigger a prune")
	}

	cache.Set(1000, 1000)
	if cache.Len() != 91 {
		t.Error("Adding one more key should have triggered a prune")
	}

	if value, ok := cache.Get(0); !ok || value!=500 {
		t.Error("Updating a key value didn't refresh its age")
	}

	cache.Close()
}



// Test Set prunes the oldest elements when there isn't space
func TestSetPrune(t *testing.T) {	
	
	cache := NewLRUCache(100, 5)
	for i:=0; i<100; i++ {
		cache.Set(i, strconv.Itoa(i))
	}
	if cache.Len() != 100 {
		t.Error("The cache cache was pruned before filling")
	}

	cache.Set(1000, 1000)
	if cache.Len() != 96 {
		t.Error("Set didn't prune the cache when it reached max size")
	}

	// Oldest elements pruned
	if (cache.Contains(0) || cache.Contains(1) || cache.Contains(2) || 
		cache.Contains(3) || cache.Contains(4)) {
		t.Error("Set didn't purge the oldest elements")
	}

	// Newest elements still in cache
	if !cache.Contains(99) || !cache.Contains(1000) {
		t.Error("Set deleted the newest elements")
	}
}


// Test Remove basic operation
func TestRemove(t *testing.T) {
	cache := NewLRUCache(100, 10)
	cache.Set("1", 1)
	cache.Set("2", 2)

	cache.Get("1")
	cache.Get("3")

	// Remove non-existent key
	cache.Remove("4")
	if cache.Len() != 2 {
		t.Error("Removed a non-existent key")
	}

	// Check key is deleted
	cache.Remove("2")
	if _, ok := cache.Get("2"); ok {
		t.Error("Remove didn't delete the key")
	}

	// Check stats left unchanged
	if hit, miss := cache.Stats(); hit != 1 || miss != 2 {
		t.Error("Remove modified stats")
	}

	cache.Close()
}


// Test Peek basic operation
func TestPeek(t *testing.T) {

	cache := NewLRUCache(100, 10)
	for i:=0; i<100; i++ {
		cache.Set(i, i)
	}
	
	// Test doesn't update stats or refresh key cache access
	// If this were a Get request it would refresh the key
	if value, ok := cache.Peek(0); !ok || value != 0 {
		t.Error("Peek returned unexpected value")
	}

	// Adding another key will cause a prune, if peek refreshed the 
	// key it should remain in the cache, otherwise it should be pruned
	cache.Set(1000, 1000)

	if _, ok := cache.Peek(0); ok {
		t.Error("Peek refreshed the age of the key")
	}
	
	// Check peek doesn't update stats
	hit, miss := cache.Stats()
	if _, ok := cache.Peek(50); !ok{
		t.Error("Unexpected Error")
	}
	if _, ok := cache.Peek("unknown"); ok{
		t.Error("Unexpected Error")
	}


	if new_hit, new_miss := cache.Stats() ; new_hit != hit || new_miss != miss {
		t.Error("Peek updated cache hit/miss stats")
	}

	cache.Close()
}


// Test Contains basic operation
func TestContains(t *testing.T) {
	cache := NewLRUCache(100, 10)
	for i:=0; i<100; i++ {
		cache.Set(i, i)
	}
	
	// If this were a Get request it would refresh the key
	if ok := cache.Contains(0); !ok {
		t.Error("Contains returned unexpected value")
	}

	// Adding another key will cause a prune, if contains refreshed the 
	// key it should remain in the cache, otherwise it should be pruned
	cache.Set(1000, 1000)

	if ok := cache.Contains(0); ok {
		t.Error("Contains refreshed the age of the key")
	}
	
	// Check containse doesn't update stats
	hit, miss := cache.Stats()
	if ok := cache.Contains(50); !ok{
		t.Error("Unexpected Error")
	}
	if ok := cache.Contains("unknown"); ok{
		t.Error("Unexpected Error")
	}

	if new_hit, new_miss := cache.Stats() ; new_hit != hit || new_miss != miss {
		t.Error("Contains updated cache hit/miss stats")
	}

	cache.Close()
}



func TestRemoveOldest(t *testing.T) {	
	cache := NewLRUCache(100, 10)
	
	// Removing from empty cache shouldn't
	cache.RemoveOldest()



	for i:=0; i<100; i++ {
		cache.Set(i, i)
	}

	if !cache.Contains(0) {
		t.Error("Cache should contain 0")
	}

	cache.RemoveOldest()
	
	if cache.Contains(0) {
		t.Error("PopOldest didn't remove 0 from cache")
	}
	
}

// Test stat generation
func TestStats(t *testing.T) {

	cache := NewLRUCache(100, 1)
	
	if hit, miss := cache.Stats(); hit!=0 || miss!=0 {
		t.Error(fmt.Sprintf("Initial stats -> hits: %v miss: %v", hit, miss))
	}

	//Test it update stats
	cache.Set(1, 1)
	cache.Set(2, 2)

	cache.Get(1)
	if hit, miss := cache.Stats(); hit!=1 || miss!=0 {
		t.Error("Stats are not accurate")
	}

	cache.Get(10)
	if hit, miss := cache.Stats(); hit!=1 || miss!=1 {
		t.Error("Stats are not accurate")
	}

	// Test purge doesn't zero stats
	cache.Purge()
	if hit, miss := cache.Stats(); hit!=1 || miss!=1 {
		t.Error("Purge shouldn't have reseted the stats")
	}
	
	cache.Close()
}


func TestResetStats(t *testing.T) {
	cache := NewLRUCache(100, 1)
		

	// Initialize status
	cache.Set(1, 1)
	cache.Set(2, 2)
	cache.Get(1)
	cache.Get(3)

	if hit, miss := cache.Stats(); hit!=1 || miss!=1 {
		t.Error("Stats should have been hit:1 miss: 1")
	}

	cache.ResetStats()
	if hit, miss := cache.Stats(); hit!=0 || miss!=0 {
		t.Error("ResetStats failed")
	}

	cache.Close()
}

func TestString(t *testing.T) {
	cache := NewLRUCache(100, 1)
	fmt.Sprintf("%v", cache)

	cache.Close()
}


func TestConcurrency(t *testing.T) {	

	// Initialize cache
	cache := NewLRUCache(100, 1)

	for i:=0; i<100; i++ {
		cache.Set(i, i)
	}

	// Functions to Set/Get cache values concurrently
	concurrentGet := func(cache *LRUCache, key interface{}, delay time.Duration) {
		time.Sleep(delay*time.Millisecond)
		cache.Get(key)
	}
	concurrentPeek := func(cache *LRUCache, key interface{}, delay time.Duration) {
		time.Sleep(delay*time.Millisecond)
		cache.Peek(key)
	}
	concurrentSet := func(cache *LRUCache, key interface{}, value interface{}, delay time.Duration) {
		time.Sleep(delay*time.Millisecond)
		cache.Set(key, value)
	}

	// Spawn routines updating/creating/reading
	for i := 0 ; i < 50; i++ {
		go concurrentGet(cache, i, 5)
		go concurrentPeek(cache, i, 3)
		go concurrentSet(cache, i, i+10, 8)
		go concurrentGet(cache, i, 4)
		go concurrentPeek(cache, i, 10)
		go concurrentSet(cache, i+100, i+200, 12)
	}

	// Wait enough time to assure all operations have finished
	time.Sleep(2000*time.Millisecond)

	// Check all cache values were updated correctly
	for i:= 0; i < 50; i++ {
		if value, ok := cache.Get(i); !ok || value.(int) != i+10 {
			t.Error("Cache not updated successfully")
		}
		if value, ok := cache.Get(i+100); !ok || value.(int) != i+200 {
			t.Error("Cache not updated successfully")
		}
	}

	cache.Close()
}



















































