package lru

import (
	"container/list"
	"log"
	"time"
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

	// expire time map
	expires map[string] time.Time

	// default expire time
	expireTime time.Duration

	// 回调函数
	OnEvicted func(key string, value Value)
}

// Real data that stored in cache
type entry struct {
	key   string
	visit int
	value Value
}

type Value interface{
	Len() int
}

func newBaseCache(maxBytes int64, OnEvicted func(string, Value)) *baseCache {
	return &baseCache{
		maxBytes: maxBytes,
		ll: list.New(),
		cache: make(map[string]*list.Element),
		expires: make(map[string]time.Time),
		expireTime: 2000 * time.Millisecond, // 2s
		OnEvicted: OnEvicted,
	}
}

// set expire time
func (bc *baseCache) SetExpireTime(expireTime time.Duration) {
	bc.expireTime = expireTime
}


// Get
func (bc *baseCache) Get(key string) (value Value, ok bool) {

	if (key == "") {
		log.Println("Get: key == nil")
	}

	// Is it expire?
	if bc.expires != nil {
		if expire, ok := bc.expires[key]; ok && !expire.IsZero(){
			if time.Now().After(expire) {
				// remove
				log.Println("Expired!")
				bc.Remove(key)
				return nil, false
			}
		}
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
		if bc.OnEvicted != nil {
			bc.OnEvicted(kv.key, kv.value)
		}
	}
}

func (bc *baseCache) AddWithExpire(key string, value Value, expire time.Duration) {
	if bc.cache == nil {
		bc.cache = make(map[string]*list.Element)
	}
	if bc.expires == nil {
		bc.expires = make(map[string]time.Time)
	}
	// expire
	if expire > 0 {
		bc.expires[key] = time.Now().Add(expire)
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

func (bc *baseCache) Add(key string, value Value) {
	bc.AddWithExpire(key, value, bc.expireTime)
}

func (bc *baseCache) Len() int {
	return bc.ll.Len()
}

func (bc *baseCache) removeElement(ele *list.Element) {
	// from double-link-list
	bc.ll.Remove(ele)
	// from map
	kv := ele.Value.(*entry)
	delete(bc.cache, kv.key)
	// delete from expires
	delete(bc.expires, kv.key)
	// resize
	bc.usedBytes -= int64(len(kv.key) + kv.value.Len())
}

func (bc *baseCache) Remove(key string) {
	if ele, ok := bc.cache[key]; ok {
		bc.removeElement(ele)
		kv := ele.Value.(*entry)
		if bc.OnEvicted != nil {
			bc.OnEvicted(kv.key, kv.value)
		}
	}
}

// remove expire
func (bc *baseCache)  cleanExpired() {
	now := time.Now()
	for key, exp := range bc.expires {
		if now.After(exp) { // remove
			log.Println("basecache.go: Auto remove expired cache!")
			bc.Remove(key)
		}
	}
}