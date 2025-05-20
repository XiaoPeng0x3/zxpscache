// grpc server

package mygroupcache

import (
	"context"
	"fmt"
	"log"
	pb "my_groupcache/cachepb"
	"my_groupcache/consistenthash"
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
	peers *consistenthash.Map
	client map[string] *client
}

const (
	defaultAddr     = "127.0.0.1:8090"
	defaultReplicas = 50
)

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
		log.Fatalf("start service FAIL!, ip = %s, error: %s", p.addr, err.Error())
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

func (p *GRPCPool) SetPeers(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	if p.client == nil {
		p.client = make(map[string]*client)
	}
	// 创建客户端
	for _, peer := range peers {
		p.client[peer] = &client{name: peer + p.addr}
	}
}

func (p *GRPCPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	log.Printf("PickPeer %s", p.peers.Get(key))
	if peer := p.peers.Get(key); peer != "" && peer != p.addr {
		log.Printf("Pick peer: %s", peer)
		return p.client[peer], true
	}
	return nil, false
}