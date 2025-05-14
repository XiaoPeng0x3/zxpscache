package lru

import (
	"fmt"
	"testing"
)


func TestLRUKBasic(t *testing.T) {
	k := 2
	maxBytes := 1024
	evicted := make(map[string]string)

	lru := newCache(k, maxBytes, func(key string, value Value) {
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
