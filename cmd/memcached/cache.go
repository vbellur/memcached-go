//LRU cache management for memcached
package main

import (
	"container/list"
	"errors"
	"sync"
)

//LRU cache, implemented with a map & doubly linked list
type lru struct {
	m       map[string]*list.Element
	dll     *list.List
	lock    *sync.RWMutex
	maxKeys int
}

// node used in doubly linked list
type node struct {
	key     string
	value   string
	flags   int
	noreply bool
	exptime int64
}

// LRU cache for key,value tuples
var cache *lru

// maximum capacity for LRU
const maxCapacity = 1024 * 1024

// Initializes data structures associated with cache
func CacheInit(capacity int) error {
	if capacity <= 0 || capacity > maxCapacity {
		err := errors.New("Invalid capacity specified for cache")
		return err
	}
	cache = new(lru)
	cache.m = make(map[string]*list.Element)
	cache.dll = new(list.List)
	cache.lock = new(sync.RWMutex)
	cache.maxKeys = capacity
	return nil
}

// Evict least recently element from cache. Element at the tail of the
// doubly linked list gets evicted. This function should only be called from
// _checkPruneCache().
func _pruneCache() error {

	e := cache.dll.Back()
	n := e.Value.(*node)
	_ = cache.dll.Remove(e)
	delete(cache.m, n.key)

	return nil

}

// Check and evict least recently used key from cache. The caller of the
// function needs to hold a lock on the cache before invoking _checkPruneCache().
func _checkPruneCache() error {
	l := len(cache.m)
	if cache.maxKeys != 0 && l == cache.maxKeys {
		return _pruneCache()
	}

	return nil
}

//Updates key if it already exists in cache. If not, a new key is inserted into
//the cache
func Upsert(key string, val string) error {

	cache.lock.Lock()
	e, ok := cache.m[key]
	if !ok {
		err := _checkPruneCache()
		if err != nil {
			cache.lock.Unlock()
			return err
		}
		n := new(node)
		n.key = key
		n.value = val
		e = cache.dll.PushFront(n)
		cache.m[key] = e
	} else {
		n := e.Value.(*node)
		n.value = val
		cache.dll.MoveToFront(e)
	}
	cache.lock.Unlock()

	return nil
}

//Retrieve value for key. If key does not exist, an error is returned.
func Get(key string) (string, error) {

	val := ""
	var err error

	cache.lock.Lock()
	e, ok := cache.m[key]
	if !ok {
		cache.lock.Unlock()
		err = errors.New("No such key")
	} else {
		n := e.Value.(*node)
		val = n.value
		cache.dll.MoveToFront(e)
		cache.lock.Unlock()
		err = nil
	}

	return val, err
}
