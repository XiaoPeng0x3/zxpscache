// cache_test.go
package mygroupcache

import (
	"testing"
	"strconv"
	"sync"
	"time"
)

func TestCacheAddAndGet(t *testing.T) {
	c := &cache{
		maxBytes: 1024,
		k:        2, // 至少访问2次才进入真正cache
	}

	key := "testKey"
	val := ByteView{b : []byte("testValue")}

	// Add key
	c.add(key, val)

	// First Get (from history)
	v, ok := c.get(key)
	if !ok || string(v.ByteSlice()) != "testValue" {
		t.Fatalf("expected value = %s, got %v (ok=%v)", "testValue", v, ok)
	}

	// Second Get (should be promoted to real cache)
	v, ok = c.get(key)
	if !ok || string(v.ByteSlice()) != "testValue" {
		t.Fatalf("expected value = %s after second get, got %v (ok=%v)", "testValue", v, ok)
	}

	// Third Get (must come from main cache now)
	v, ok = c.get(key)
	if !ok || string(v.ByteSlice()) != "testValue" {
		t.Fatalf("expected value = %s from cache, got %v (ok=%v)", "testValue", v, ok)
	}
}

func TestCacheUpdate(t *testing.T) {
	c := &cache{
		maxBytes: 1024,
		k:        1,
	}

	key := "updateKey"
	val1 := ByteView{[]byte("val1")}
	val2 := ByteView{[]byte("val2")}

	c.add(key, val1)
	v, _ := c.get(key)
	if string(v.ByteSlice()) != "val1" {
		t.Fatalf("expected val1, got %s", v.ByteSlice())
	}

	c.add(key, val2)
	v, _ = c.get(key)
	if string(v.ByteSlice()) != "val2" {
		t.Fatalf("expected val2 after update, got %s", v.ByteSlice())
	}
}

func TestCacheConcurrency(t *testing.T) {
	c := &cache{
		maxBytes: 2 << 15,
		k:        2,
	}

	var wg sync.WaitGroup
	n := 1000

	// 并发写入
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := "key" + strconv.Itoa(i)
			val := ByteView{[]byte("val" + strconv.Itoa(i))}
			c.add(key, val)
		}(i)
	}

	// 并发读取（访问两次才进真实 cache）
	for i := 0; i < n; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			key := "key" + strconv.Itoa(i)
			c.get(key)
		}(i)
		go func(i int) {
			defer wg.Done()
			key := "key" + strconv.Itoa(i)
			c.get(key)
		}(i)
	}

	wg.Wait()

	// 随机 spot check 几个 key 是否能获取成功
	for i := 0; i < n; i++ {
		key := "key" + strconv.Itoa(i)
		val, ok := c.get(key)
		if !ok {
			t.Errorf("key %s not found after concurrent access", key)
		}
		if val.Len() == 0 {
			t.Errorf("key %s returned empty value", key)
		}
	}
}

// 后台清理测试（需要配合 lru.Cache 中 CleanExpired 实现）
func TestEvictionLoop(t *testing.T) {
	c := &cache{
		maxBytes: 2 << 15,
		k:        2,
	}

	c.add("key1", ByteView{b: []byte("123")})
	c.add("key2", ByteView{b: []byte("456")})

	if val, ok := c.get("key1"); !ok || string(val.ByteSlice()) != "123" {
		t.Fatalf("val should be 123 but be %v", val)
	}

	c.startEvictionLoop(1000 * time.Millisecond)

	// 统计是否删除
	time.Sleep(3 * time.Second)
	if _, ok := c.get("key1"); ok {
		t.Fatalf("Cache expires")
	}
	if _, ok := c.get("key2"); ok {
		t.Fatalf("Cache expires")
	}

	// 没有 panic 表示后台清理逻辑没有死锁
	t.Log("eviction loop ran safely")
}

