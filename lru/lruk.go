// 实现lruk
package lru


type Cache struct {
	// history
	history *baseCache

	// cache
	cache *baseCache

	// maxBytes
	maxBytes int

	// k
	k int
}

func newCache(k int, maxBytes int, OnEvicted func(string, Value)) *Cache{
	history := newBaseCache(int64(maxBytes), OnEvicted)
	cache := newBaseCache(int64(maxBytes), OnEvicted)
	return &Cache{
		history: history,
		cache: cache,
		maxBytes: maxBytes,
		k: k,
	}
}

// Get
func (c *Cache) Get(key string) (value Value, ok bool) {
	// from cache
	if value, ok := c.cache.Get(key); ok {
		return value, ok
	}

	// from history
	if value, ok := c.history.Get(key); ok {
		// visit += 1
		ele := c.history.cache[key]
		kv := ele.Value.(*entry)
		// visit +=1
		kv.visit += 1

		if kv.visit >= c.k {
			// add to cache
			c.cache.Add(kv.key, kv.value)
			c.history.Remove(kv.key)
		} else {
			c.history.ll.MoveToFront(ele)
		}
		return value, ok
	}
	return
}

func (c *Cache) Add(key string, value Value) {
	// in cache
	if _, ok := c.cache.cache[key]; ok {
		c.cache.Add(key, value)
		return
	}

	// in history
	if ele, ok := c.history.cache[key]; ok {
		kv := ele.Value.(*entry)
		kv.value = value
		kv.visit += 1
		// in cache
		if kv.visit >= c.k {
			c.history.Remove(key)
			c.cache.Add(key, value)
		} else {
			c.history.Add(key, value)
		}
		return
	} else {
		c.history.Add(key, value)
		ele := c.history.cache[key]
		kv := ele.Value.(*entry)
		kv.visit += 1
		return
	}

}

func (c *Cache) Remove(key string) {
	c.history.Remove(key)
	c.cache.Remove(key)
}

func (c *Cache) RemoveOldest() {
	c.cache.RemoveOldest()
}


