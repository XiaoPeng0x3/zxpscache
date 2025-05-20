package mygroupcache

import (
	"fmt"
	"log"
	pb "my_groupcache/cachepb"
	"my_groupcache/singleflight"
	"sync"
)

// 回调函数
type Getter interface {
	Get(string) ([]byte, error)
}

// 函数适配器
type GetterFunc func(string) ([]byte, error)

func (f GetterFunc) Get(key string) (bytes []byte, err error) {
	return f(key)
}

// TODO single-flight
// A Group is a cache namespace and associated data loaded spread over
type Group struct {
	name      string
	getter    Getter
	mainCache cache
	peers PeerPicker
	// singleflight
	loader *singleflight.Group
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

// NewGroup create a new instance of Group
func NewGroup(name string, maxBytes int, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{maxBytes: maxBytes},
		loader: &singleflight.Group{},
	}
	groups[name] = g
	return g
}

// GetGroup returns the named group previously created with NewGroup, or
// nil if there's no such group.
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// RegisterPeers registers a PeerPicker for choosing remote peer
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// Get value for a key from cache
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	// 本地调用
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}

	return g.load(key)
}

// 改造为调用远程结点 + 本地调用
func (g *Group) load(key string) (value ByteView, err error) {
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			// pick peer
			// log.Println(g.peers)
			// log.Println(g.peers.PickPeer(key))
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err := g.getFromPeer(peer, key); err == nil {
					log.Printf("Remote peers from %s", g.name)
					return value, nil
				}
			}
		}
		return g.getLocally(key)
	})
	
	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	// 调用客户端的Get
	request := &pb.Request{
		Group: g.name,
		Key: key,
	}
	response := &pb.Response{}
	err := peer.Get(request, response)
	if err != nil {
		return ByteView{}, fmt.Errorf("Error: %s", err.Error())
	}
	return ByteView{b: response.Value}, nil
}

func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err

	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	log.Print("From db!")
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}