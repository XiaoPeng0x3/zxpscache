// from geektutu
package mygroupcache

import (
	"fmt"
	"log"
	"sync"
	"testing"
	"time"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
	"Nami": "679",
}

func creatGroup() *Group {
	return NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[slowDB] search key: " + key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

func TestGroupServer(t *testing.T) {
	addrMap := map[int]string{
		8001: "127.0.0.1:8001",
		8002: "127.0.0.1:8002",
		8003: "127.0.0.1:8003",
	}
	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}
	groups := make([]*Group, 0)
	var mu sync.Mutex

	// 启动多个 gRPC 节点
	for port := range addrMap {
		go func(port int) {
			pool := NewGRPCPool(addrMap[port], 0, nil) // 将 group 传入
			pool.SetPeers(addrs...)
			// 创建缓存组
			group := creatGroup()
			group.RegisterPeers(pool)
			mu.Lock()
			groups = append(groups, group)
			mu.Unlock()
			if err := pool.Start(); err != nil {
				log.Fatalf("gRPC server on %s failed: %v", addrMap[port], err)
				return
			}
			select{}
		}(port)
	}

	// 等待服务启动
	time.Sleep(2 * time.Second)
	g := groups[0]
	log.Printf("group: %v", g)
	g.Get("Jack")
	g.Get("Tom")
	g.Get("Tom")
	
}
