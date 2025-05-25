// consistent hash
package consistenthash

import (
	"hash/crc32"
	"log"
	"sort"
	"strconv"
	"sync"
)

// Hash maps bytes to uint32
type Hash func(data []byte) uint32

// Map constains all hashed keys
type Map struct {
	rwLock sync.RWMutex
	hash     Hash
	replicas int
	keys     []int // Sorted
	hashMap  map[int]string
}

// New creates a Map instance
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add adds some keys to the hash.
func (m *Map) Add(keys ...string) {
	m.rwLock.Lock()
	defer m.rwLock.Unlock()
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

// Get gets the closest item in the hash to the provided key.
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}
	m.rwLock.RLock()
	defer m.rwLock.RUnlock()
	hash := int(m.hash([]byte(key)))
	// Binary search for appropriate replica.
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	s := m.hashMap[m.keys[idx%len(m.keys)]]
	log.Printf("选择结点为: %s", s)
	return m.hashMap[m.keys[idx%len(m.keys)]]
}

// Remove use to remove a key and its virtual keys on the ring and map
func (m *Map) Remove(key string) {
	m.rwLock.Lock()
	defer m.rwLock.Unlock()
	for i := 0; i < m.replicas; i++ {
		hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
		idx := sort.SearchInts(m.keys, hash)
		if idx < len(m.keys) && m.keys[idx] == hash {
			m.keys = append(m.keys[:idx], m.keys[idx+1:]...)
			delete(m.hashMap, hash)
		}
	}
}