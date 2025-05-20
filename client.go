package mygroupcache

import (
	"context"
	"fmt"
	"log"
	pb "my_groupcache/cachepb"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type client struct{
	name string
	grpcClient pb.CacheServiceClient
}

// 实现接口
func (c *client) Get(in *pb.Request, out *pb.Response) error {

	// 建立连接
	conn, err := grpc.NewClient(c.name, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}


	// 构造context
	ctx, cancle := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancle()
	if c.grpcClient == nil {
		c.grpcClient = pb.NewCacheServiceClient(conn)
	}
	out, err = c.grpcClient.Get(ctx, in)
	if err != nil {
		return fmt.Errorf("grpc client Get() error: %v", err)
	}
	return nil
}