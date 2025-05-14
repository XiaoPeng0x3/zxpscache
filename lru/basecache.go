package lru

import (
	"container/list"
	"log"
)
type baseCache struct {
	// Allowed MaxBytes
	maxBytes int64

	// Has used Bytes
	usedBytes int64

	// double link list
	ll    *list.List

	// for O(1) Get
	cache map[string]*list.Element
}

// Real data that stored in cache
type entry struct {
	key   string
	value Value
}

type Value interface{
	Len() int
}

func newBaseCache(maxBytes int64) *baseCache {
	return &baseCache{
		maxBytes: maxBytes,
		ll: list.New(),
		cache: make(map[string]*list.Element),
	}
}


// Get
func (bc *baseCache) Get(key string) (value Value, ok bool) {

	if (key == "") {
		log.Println("Get: key == nil")
	}

	// Get from cache
	if ele, ok := bc.cache[key]; ok { // cache hit
		log.Printf("lru.go: cache hit!")
		// get
		bc.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return 
}

func (bc *baseCache) RemoveOldest() {
	ele := bc.ll.Back()
	if ele != nil {
		// from map
		kv := ele.Value.(*entry)
		delete(bc.cache, kv.key)

		// from double-link-list
		bc.ll.Remove(ele)
		log.Printf("RemoveOldest cache: key= %s, value= %v\n", kv.key, kv.value)

		// resize
		bc.usedBytes -= int64(len(kv.key) + kv.value.Len())
	}
}

func (bc *baseCache) Add(key string, value Value) {
	if bc.cache == nil {
		bc.cache = make(map[string]*list.Element)
	}
	if ele, ok := bc.cache[key]; ok {
		// update 
		// from double-link-list
		bc.ll.MoveToFront(ele)
		// resize
		kv := ele.Value.(*entry)
		bc.usedBytes += int64(value.Len() - kv.value.Len())
		// value
		kv.value = value
	} else {
		// linklist
		ele := bc.ll.PushFront(&entry{key: key, value: value})
		// map
		bc.cache[key] = ele
		// resize
		bc.usedBytes += int64(len(key) + value.Len())
	}
	// drop cache
	for bc.maxBytes != 0 && bc.maxBytes < bc.usedBytes {
		bc.RemoveOldest()
	}
}

func (bc *baseCache) Len() int {
	return bc.ll.Len()
}