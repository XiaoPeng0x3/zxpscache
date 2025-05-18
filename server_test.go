package mygroupcache

import (
	"fmt"
	"log"
	_ "sync"
	"testing"

	"google.golang.org/grpc"
	pb "my_groupcache/cachepb"
	"context"
	"time"
)

func TestServerGet(t *testing.T) {
	NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
	// 创建grpc服务
	s := NewGRPCPool("localhost:50001", "zxp")
	// 注册
	go func() {
		err := s.Start()
		if err != nil {
			log.Fatal("New Service Fail")
			return
		}
	}()

	// 创建一个客户端
	// 创建一个客户端连接（连接到你刚才启动的服务端）
	conn, err := grpc.Dial("localhost:50001", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	// 创建客户端实例
	client := pb.NewCacheServiceClient(conn)

	// 设置请求参数
	req := &pb.Request{
		Group: "scores",
		Key:   "Tom",
	}

	// 发起调用
	ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
	defer cancel()

	resp, err := client.Get(ctx, req)
	if err != nil {
		t.Fatalf("could not get: %v", err)
	}

	t.Logf("Got response: %s", string(resp.Value))
	
}