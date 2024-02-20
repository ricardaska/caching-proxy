package caching

import (
	"sync"
	"time"
)

const EVICT_EXPIRED_AFTER = 3 * 60 * 1000

type DataLoader[T any] func() T

type CacheObject[T any] struct {
	sync.Mutex
	loaded    bool
	data      T
	expiresAt int64
}

type Cache[T any] struct {
	sync.Mutex
	data           map[string]*CacheObject[T]
	ttl            int64
	evicting       bool
	evictExpiredAt int64
}

func NewCache[T any](ttl time.Duration) *Cache[T] {
	return &Cache[T]{
		data: make(map[string]*CacheObject[T]),
		ttl:  ttl.Milliseconds(),
	}
}

func (cache *Cache[T]) Store(key string, value T) {
	cache.Lock()
	defer cache.Unlock()
	now := time.Now().UnixMilli()

	object := &CacheObject[T]{
		expiresAt: cache.ttl + now,
		data:      value,
	}
	cache.data[key] = object
}

func (cache *Cache[T]) Remove(key string) (stored T) {
	cache.Lock()
	defer cache.Unlock()

	if value, ok := cache.data[key]; ok {
		delete(cache.data, key)
		stored = value.data
	}
	return
}

func (cache *Cache[T]) RemoveKeyFunc(remove func(key string) bool) {
	cache.Lock()
	defer cache.Unlock()

	for key := range cache.data {
		if remove(key) {
			delete(cache.data, key)
		}
	}
}

func (cache *Cache[T]) Load(key string, loadData DataLoader[T]) T {
	cache.Lock()
	now := time.Now().UnixMilli()

	if cache.evictExpiredAt < now && !cache.evicting {
		cache.evictExpiredAt = now + EVICT_EXPIRED_AFTER
		cache.evicting = true
		go cache.evictExpired()
	}

	var (
		object *CacheObject[T]
		ok     bool
	)

	if object, ok = cache.data[key]; !ok || object.expiresAt < now {
		object = &CacheObject[T]{
			expiresAt: cache.ttl + now,
		}
		cache.data[key] = object

		object.Lock()
		defer object.Unlock()

		cache.Unlock()

		object.data = loadData()
		return object.data
	}

	cache.Unlock()

	if !object.loaded {
		object.Lock()
		defer object.Unlock()
	}
	return object.data
}

func (cache *Cache[T]) evictExpired() {
	cache.Lock()
	defer cache.Unlock()

	cache.evicting = false

	now := time.Now().UnixMilli()
	for key, value := range cache.data {
		if !value.loaded || value.expiresAt == -1 {
			continue
		}

		if value.expiresAt > now {
			continue
		}

		delete(cache.data, key)
	}
}
