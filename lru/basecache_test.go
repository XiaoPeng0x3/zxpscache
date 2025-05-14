package lru

import (
	"log"
	"testing"
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