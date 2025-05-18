// grpc server

package mygroupcache

import (
	"context"
	"fmt"
	"log"
	pb "my_groupcache/cachepb"
	"net"
	"sync"

	"time"

	"google.golang.org/grpc"
)

type GRPCPool struct {
	*pb.UnimplementedCacheServiceServer
	addr string // 服务地址
	svcName string // 服务名称
	grpcServer *grpc.Server
	mu sync.Mutex
}

// ServerOptions 服务器配置选项
type ServerOptions struct {
	EtcdEndpoints []string      // etcd端点
	DialTimeout   time.Duration // 连接超时
	MaxMsgSize    int           // 最大消息大小
	TLS           bool          // 是否启用TLS
	CertFile      string        // 证书文件
	KeyFile       string        // 密钥文件
}

// DefaultServerOptions 默认配置
var DefaultServerOptions = &ServerOptions{
	EtcdEndpoints: []string{"localhost:2379"},
	DialTimeout:   5 * time.Second,
	MaxMsgSize:    4 << 20, // 4MB
}

// ServerOption 定义选项函数类型
type ServerOption func(*ServerOptions)

func NewGRPCPool(addr string, svcName string) *GRPCPool {
	return &GRPCPool{
		addr: addr,
		svcName: svcName,
		grpcServer: grpc.NewServer(),
	}

}
// Log info with server name
func (p *GRPCPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.addr, fmt.Sprintf(format, v...))
}

func (p *GRPCPool) Get(ctx context.Context, req *pb.Request) (response *pb.Response, err error) {

	group_name := req.GetGroup()
	key_name := req.GetKey()

	group := GetGroup(group_name)
	if group == nil {
		log.Fatalf("No such group : %s", group_name)
		return
	}
	mu.Lock()
	defer mu.Unlock()
	view, err := group.Get(key_name)
	if err != nil {
		log.Fatal("Can't get data")
		return
	}
	return &pb.Response{
		Value: view.ByteSlice(),
	}, nil
}

// start
func (p *GRPCPool) Start() error {
	lis, err := net.Listen("tcp", p.addr)
	if err != nil {
		log.Fatalf("start service FAIL!, ip = %s", p.addr)
	}
	if p.grpcServer == nil {
		p.grpcServer = grpc.NewServer()
	}
	// 注册
	pb.RegisterCacheServiceServer(p.grpcServer, p)
	return p.grpcServer.Serve(lis)
}

func (p *GRPCPool) Stop() {
	p.grpcServer.GracefulStop()
}