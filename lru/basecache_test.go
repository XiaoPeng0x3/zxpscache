package lru

import (
	"log"
	"testing"
	"time"
)

type String string

func (s String) Len() int{
	return len(s)
}

// 测试Get
func TestGet(t *testing.T) {
	// 添加几条数据
	baseCache := newBaseCache(int64(10), nil)
	baseCache.Add("zxp", String("18"))
	log.Println(baseCache)

	// 测试Get
	if v, ok := baseCache.Get("zxp"); !ok || string(v.(String)) != "18" {
		t.Fatalf("cache hit zxp=18 failed")
	}
	if _, ok := baseCache.Get("key2"); ok {
		t.Fatalf("cache miss key2 failed")
	}
}

func TestRemoveoldest(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "k3"
	v1, v2, v3 := "value1", "value2", "v3"
	cap := len(k1 + k2 + v1 + v2)
	lru := newBaseCache(int64(cap), nil)
	lru.Add(k1, String(v1))
	lru.Add(k2, String(v2))
	lru.Add(k3, String(v3))

	if _, ok := lru.Get("key1"); ok || lru.Len() != 2 {
		t.Fatalf("Removeoldest key1 failed")
	}
}

func TestCacheExpiration(t *testing.T) {
	var evictedKeys []string

	cache := newBaseCache(100, func(key string, value Value) {
		evictedKeys = append(evictedKeys, key)
	})

	// 添加一个1秒后过期的key
	cache.AddWithExpire("foo", String("bar"), time.Second)

	// Never expire
	cache.AddWithExpire("zxp", String("hahah"), 0)

	// 立即应命中
	if val, ok := cache.Get("foo"); !ok || string(val.(String)) != "bar" {
		t.Errorf("expected to get value 'bar', got %v, hit=%v", val, ok)
	}

	// 等待1.5秒后应 miss
	time.Sleep(1500 * time.Millisecond)

	if _, ok := cache.Get("foo"); ok {
		t.Errorf("expected cache miss after expiration, but got hit")
	}
	time.Sleep(15000 * time.Millisecond)
	if val, ok := cache.Get("zxp"); !ok || string(val.(String)) != "hahah" {
		t.Errorf("expected to get value 'hahah', got %v, hit=%v", val, ok)	
	}

	// 验证是否回调了 OnEvicted
	if len(evictedKeys) != 1 || evictedKeys[0] != "foo" {
		t.Errorf("expected 'foo' to be evicted, got %v", evictedKeys)
	}
}

func TestCleanExpired(t *testing.T) {
	var evictedKeys []string
	cache := newBaseCache(100, func(key string, value Value) {
		evictedKeys = append(evictedKeys, key)
	})

	// 添加一个10s过期的key, 在空间充足的情况下，即使cache过期，也不会发生清除
	cache.AddWithExpire("zxp", String("dsb"), 5 * time.Second)
	// sleep, 测试失效
	// time.Sleep(6 * time.Second)
	if val, ok := cache.Get("zxp"); !ok || val.(String) != "dsb" {
		t.Fatalf("val should be dsb but be %v", val)
	}
	// 测试删除缓存
	time.Sleep(6 * time.Second)
	cache.cleanExpired()
	if cache.usedBytes != 0 {
		t.Fatal("expire cache should be evicted but not!")
	}
}