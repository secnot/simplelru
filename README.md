# SimpleLRU  [![Build Status](https://travis-ci.org/secnot/simplelru.svg?branch=master)](https://travis-ci.org/secnot/simplelru) [![GoDoc](https://godoc.org/github.com/secnot/simplelru?status.svg)](http://godoc.org/github.com/secnot/simplelru) [![Go Report Card](https://goreportcard.com/badge/github.com/secnot/simplelru)](https://goreportcard.com/report/github.com/secnot/simplelru) 

A LRU Cache (Least Recently Used) written in Go.

- Concurrency-safe
- Supports auto-fetching for items on cache miss.
- 100% Test coverage


## Installation

Download the package and its only dependency:

```bash
go get github.com/secnot/orderedmap
go get github.com/secnot/simplerlu
```

## Usage

A basic LRU cache:

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

Or with a function to fetch the value from another source on cache miss:

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


// Simulated db query, with delay (concurrency-safe)
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
	value, _ := cache.Get(1) // Ignore errors
	fmt.Println(value) // > query result: 1

	// No fetch for cached keys
	cache.Set(2, "2 is cached")
	value, _ = cache.Get(2) // Ignore errors
	fmt.Println(value) // > 2 is cached
}
``` 


## Documentation

## TYPE

```go
type FetchFunc func(key interface{}) (value interface{}, ok bool)
```

Type of functions to fetch missing values when there is a miss.


## TYPE

```go
type LRUCache struct {
    // contains filtered or unexported fields
}
```

#### func NewFetchingLRUCache

```go
func NewFetchingLRUCache(size int, pruneSize int,
    fetcher FetchFunc,
    fetchWorkers uint32,
    fetchQueueSize uint32) *LRUCache
```    
	
New LRUCache with optional fetch function to retrieve keys on cache miss.

- fetchWorkers: Fetch worker pool size, if it's greater than one fetch function 
must be concurrency-safe.

- fetchQueueSize: Worker pool job queue, must be tunned for the number of workers 
and expected concurrent cache misses.


#### func NewLRUCache

```go
func NewLRUCache(size int, pruneSize int) *LRUCache
```

New LRUCache without lookup function


#### func (*LRUCache) Close

```go
func (c *LRUCache) Close()
```

Stop all fetch routines


#### func (*LRUCache) Contains

```go
func (c *LRUCache) Contains(key interface{}) bool
```

Returns true if the cache contains the key (no side-effects)


#### func (*LRUCache) Get

```go
func (c *LRUCache) Get(key interface{}) (value interface{}, ok bool)
```
    
Get the key value, if not cached use the fetch function if available.

#### func (*LRUCache) Len

```go
func (c *LRUCache) Len() (size int)
```

Return number of keys in cache


#### func (*LRUCache) Peek

```go
func (c *LRUCache) Peek(key interface{}) (value interface{}, ok bool)
```
    
Get key value without updating cache, stats, or triggering a fetch


#### func (*LRUCache) Purge

```go
func (c *LRUCache) Purge()
```

Purge all cache contents (without reseting stats). Items currently being
fetched are not purged.


#### func (*LRUCache) Remove

```go
func (c *LRUCache) Remove(key interface{})
```    

Remove key from cache


#### func (*LRUCache) RemoveOldest

```go
func (c *LRUCache) RemoveOldest()
```    

Remove Least Recently Used key from cache


#### func (*LRUCache) ResetStats

```go
func (c *LRUCache) ResetStats()
```

Reset cache stats


#### func (*LRUCache) Resize

```go
func (c *LRUCache) Resize(size int, pruneSize int)
```

Set new max cache size, if its smaller than the current size
it will be pruned to size.


#### func (*LRUCache) Set

```go
func (c *LRUCache) Set(key interface{}, value interface{}) (pruned bool)
```
    
Set or update key value, returns true if the cache was pruned to make
space for a new key. Set has priority over fetched values, so if the key
set is being fetched, all goroutines waiting will wakeup and receive the
'setted' value while the fetch results are discarded.


#### func (*LRUCache) Stats

```go
func (c *LRUCache) Stats() (hit uint64, miss uint64)
```    

Return cache stats


#### func (*LRUCache) String

```go
func (c *LRUCache) String() string
```
    
Stringer interface
