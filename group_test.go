// from geektutu
package mygroupcache

import (
	"fmt"
	"log"
	"testing"
	"time"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
	"Nami": "679",
}

func TestGet(t *testing.T) {
	loadCounts := make(map[string]int, len(db))
	gee := NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				if _, ok := loadCounts[key]; !ok {
					loadCounts[key] = 0
				}
				loadCounts[key] += 1
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	for k, v := range db {
		if view, err := gee.Get(k); err != nil || view.String() != v {
			t.Fatal("failed to get value of Tom")
		} // load from callback function
		if _, err := gee.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		} // cache hit
	}

	if view, err := gee.Get("unknown"); err == nil {
		t.Fatalf("the value of unknow should be empty, but %s got", view)
	}
}

func createGroup() *Group {
	return NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

func startCacheServer(addr string, addrs []string, gee *Group) {
	peer := NewGRPCPool(addr, "zxp")
	peer.SetPeers(addrs...)
	gee.RegisterPeers(peer)
	// 启动服务
	peer.Start()
}

func TestCache(t *testing.T) {
	addrMap := map[int]string{
		50001: "localhost:50001",
		50002: "localhost:50002",
		50003: "localhost:50003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	// 开启服务
	go func() {
		for port, _ := range addrMap {
			g := createGroup()
			startCacheServer(addrMap[port], addrs, g)
		}
	}()
	time.Sleep(2 * time.Second)

	// 客户端获取
	client := NewGRPCPool("localhost:50006", "zxp")
	client.SetPeers(addrs...)

	clientGroup := createGroup()
	clientGroup.RegisterPeers(client)

	val, err := clientGroup.Get("Tom")
	if string(val.ByteSlice()) != "630" || err != nil {
		log.Fatal("Error")
	}

	clientGroup.Get("Sam")

	clientGroup.Get("Sam")
} 