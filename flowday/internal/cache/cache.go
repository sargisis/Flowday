package cache

import (
	"sync"
	"time"
)

var (
	cacheMap sync.Map
	mu       sync.RWMutex
)

type cacheItem struct {
	value      interface{}
	expiration time.Time
}

func Init() {
	// Initialize in-memory cache
	// This is a simple implementation - can be extended with Redis support
}

func Get(key string) (interface{}, bool) {
	mu.RLock()
	defer mu.RUnlock()
	
	item, ok := cacheMap.Load(key)
	if !ok {
		return nil, false
	}
	
	cacheItem := item.(cacheItem)
	if time.Now().After(cacheItem.expiration) {
		cacheMap.Delete(key)
		return nil, false
	}
	
	return cacheItem.value, true
}

func Set(key string, value interface{}, ttl time.Duration) {
	mu.Lock()
	defer mu.Unlock()
	
	cacheMap.Store(key, cacheItem{
		value:      value,
		expiration: time.Now().Add(ttl),
	})
}

func Delete(key string) {
	mu.Lock()
	defer mu.Unlock()
	cacheMap.Delete(key)
}

func Clear() {
	mu.Lock()
	defer mu.Unlock()
	cacheMap.Range(func(key, value interface{}) bool {
		cacheMap.Delete(key)
		return true
	})
}
