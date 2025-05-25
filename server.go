// grpc server

package mygroupcache

import (
	"context"
	"fmt"
	"log"
	pb "my_groupcache/cachepb"
	"my_groupcache/consistenthash"
	"my_groupcache/registry"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type GRPCPool struct {
	pb.UnimplementedCacheServiceServer
	addr string // 服务地址
	status bool // 是否启动
	stopSignal chan error // 通知registry revoke服务
	replicas int                     // 一致性哈希时，key 翻倍的倍数。如果为空，则默认为 50
	hashFunc consistenthash.Hash
	mu sync.Mutex
	peers *consistenthash.Map
	client map[string] *client
}

const (
	defaultAddr     = "127.0.0.1:8090"
	defaultReplicas = 50
)

func NewGRPCPool(addr string, replicas int, hashFunc consistenthash.Hash) *GRPCPool {
	if addr == "" {
		addr = defaultAddr
	}
	if replicas == 0 {
		replicas = defaultReplicas
	}
	s := &GRPCPool{
		addr: addr,
		replicas: replicas,
		hashFunc: hashFunc,
	}
	RegisterPeerPicker(s)
	return s

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
	response = &pb.Response{}
	// 压缩序列化
	response.Value, err = proto.Marshal(&pb.Response{Value: view.ByteSlice()})
	if err != nil {
		log.Fatalf("Marshal err : %s", err.Error())
		return
	}
	return
}

// start
func (p *GRPCPool) Start() error {
	p.mu.Lock()
	if p.status {
		p.mu.Unlock()
		return fmt.Errorf("server can only be started once")
	}
	p.status = true
	p.stopSignal = make(chan error)
	p.mu.Unlock()

	// 建议直接使用完整的监听地址
	lis, err := net.Listen("tcp", p.addr)
	if err != nil {
		log.Printf("failed to listen: %v", err)
		return err
	}

	gs := grpc.NewServer()
	pb.RegisterCacheServiceServer(gs, p)

	// 注册服务到 etcd（异步，不影响服务启动）
	go func() {
		if err := registry.RegisterServiceToETCD("groupcache", p.addr, p.stopSignal); err != nil {
			log.Printf("etcd register error: %v", err)
		}
	}()

	log.Printf("gRPC Server listening at %s", p.addr)
	if err := gs.Serve(lis); err != nil {
		log.Printf("grpc serve error: %v", err)
		return err
	}
	return nil
}


func (p *GRPCPool) SetPeers(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, p.hashFunc)
	p.peers.Add(peers...)
	p.client = make(map[string]*client, len(peers))
	// 创建客户端
	// groupcache/ip:port
	for _, peer := range peers {
		p.client[peer] = &client{name: "groupcache/" + peer}
	}
}

func (p *GRPCPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	//log.Printf("PickPeer %s", p.peers.Get(key))
	if peer := p.peers.Get(key); peer != "" && peer != p.addr {
		log.Printf("[Server: %s] : Pick Peer: %s, request key: %s", p.addr, peer, key)
		// log.Printf("Pick peer: %s", peer)
		return p.client[peer], true
	}
	return nil, false
}