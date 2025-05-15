package mygroupcache

import (
	"my_groupcache/lru"
	"sync"
	"time"
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
	// stop
	stopChan chan struct{}
	// running
	evictionRunning bool
}

// add
func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		c.lru = lru.NewCache(c.k, c.maxBytes, nil)
		go c.startEvictionLoop(60 * time.Second)
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

func (c *cache) startEvictionLoop(interval time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.evictionRunning {
		return
	}
	c.evictionRunning = true
	c.stopChan = make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.mu.Lock()
				if c.lru != nil {
					c.lru.CleanExpired()
				}
				c.mu.Unlock()
			case <-c.stopChan:
				return
			}
		}
	}()
}


func (c *cache) stopEvictionLoop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopChan != nil {
		close(c.stopChan)
		c.stopChan = nil
		c.evictionRunning = false
	}
}