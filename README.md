# SimpleLRU  [![Build Status](https://travis-ci.org/secnot/simplelru.svg?branch=master)](https://travis-ci.org/secnot/simplelru) [![GoDoc](https://godoc.org/github.com/secnot/simplelru?status.svg)](http://godoc.org/github.com/secnot/simplelru) [![Go Report Card](https://goreportcard.com/badge/github.com/secnot/simplelru)](https://goreportcard.com/report/github.com/secnot/simplelru) 

A LRU Cache (Least Recently Used) written in Go.

- Concurrency-safe
- Supports auto-fetching for items on cache miss.
- 100% Test coverage


## Installation

Download the package and its only dependency:

```bash
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

The full API documentations is available at [GoDoc](http://godoc.org/github.com/secnot/simplelru).
