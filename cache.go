package mygroupcache

import (
	"my_groupcache/lru"
	"sync"
)

type cache struct {
	// maxBytes
	maxBytes int
	// k
	k int
	// lru-k cache
	lru *lru.Cache
	// lock
	mu sync.Mutex
}

// add
func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		c.lru = lru.NewCache(c.k, c.maxBytes, nil)
	}
	c.lru.Add(key, value)
}

// get
func (c *cache) get(key string) (byteview ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return
	}
	if value, ok := c.lru.Get(key); ok {
		return value.(ByteView), ok
	}
	return
}