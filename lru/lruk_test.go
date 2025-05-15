package lru

import (
	"fmt"
	"testing"
	"time"
)


func TestLRUKBasic(t *testing.T) {
	k := 2
	maxBytes := 1024
	evicted := make(map[string]string)

	lru := NewCache(k, maxBytes, func(key string, value Value) {
		evicted[key] = string(value.(String))
	})

	// Add a new key
	lru.Add("a", String("1")) // history["a"]: visit = 1
	if _, ok := lru.cache.Get("a"); ok {
		t.Fatal("expect 'a' not in cache yet")
	}

	// Second access promotes to real cache
	lru.Get("a") // visit = 2, promote to cache
	if _, ok := lru.cache.Get("a"); !ok {
		t.Fatal("expect 'a' to be promoted to cache")
	}

	// Add b, but only 1 visit, still in history
	lru.Add("b", String("2"))
	if _, ok := lru.cache.Get("b"); ok {
		t.Fatal("expect 'b' still in history")
	}
	lru.Get("b") // second visit
	if _, ok := lru.cache.Get("b"); !ok {
		t.Fatal("expect 'b' promoted to cache")
	}

	// Check removal
	lru.Remove("a")
	if _, ok := lru.Get("a"); ok {
		t.Fatal("expect 'a' to be removed")
	}

	// Trigger eviction
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("k%d", i)
		val := String("value")
		lru.Add(key, val)
		lru.Get(key) // promote
	}
	// Should have triggered some evictions
	if len(evicted) == 0 {
		t.Fatal("expect some keys to be evicted")
	}
}

func TestLRUKAdd(t *testing.T) {
	evictedKeys := make([]string, 0)

	// k=2, maxBytes=100
	c := NewCache(2, 100, func(key string, value Value) {
		evictedKeys = append(evictedKeys, key)
	})

	// Add A,B,C
	c.Add("A", String("alpha"))  // visit=1
	c.Add("B", String("beta"))   // visit=1
	c.Add("C", String("gamma"))  // visit=1

	// A should be in history, visit count = 1
	if _, ok := c.Get("A"); !ok {
		t.Error("Expected A in history")
	}

	// A visit=2 -> promoted to cache
	if _, ok := c.Get("A"); !ok {
		t.Error("Expected A to be promoted to cache")
	}

	if _, ok := c.cache.Get("A"); !ok {
		t.Error("Expected A to be in cache")
	}
	if _, ok := c.history.Get("A"); ok {
		t.Error("Expected A to be removed from history")
	}
}

func TestLRUKEviction(t *testing.T) {
	evicted := make([]string, 0)

	c := NewCache(2, 50, func(key string, value Value) {
		evicted = append(evicted, key)
	})

	c.Add("X", String("1234567890")) // 10 bytes
	c.Add("Y", String("abcdefghij")) // 10 bytes
	c.Get("X") // visit = 2 -> promoted to cache
	c.Get("X")

	c.Add("Z", String("ZZZZZZZZZZ")) // 10 bytes, total = 30

	// Add large entry to force eviction
	c.Add("BIG", String("01234567890123456789")) // 20 bytes

	if len(evicted) == 0 {
		t.Error("Expected eviction to happen")
	}
}

func TestLRUKExpire(t *testing.T) {
	c := NewCache(2, 100, nil)
	c.history.expireTime = 500 * time.Millisecond
	c.cache.expireTime = 500 * time.Millisecond

	c.Add("temp", String("expire-soon"))

	time.Sleep(600 * time.Millisecond)

	if _, ok := c.Get("temp"); ok {
		t.Error("Expected key 'temp' to expire and be removed")
	}
}
