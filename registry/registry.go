package registry

import (
	"context"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"log"
	"time"
)

var (
	defaultEtcdConfig = clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
	}
)

// RegisterServiceToETCD 注册一个服务至etcd. 注意 Register将不会return 如果没有error的话
func RegisterServiceToETCD(serviceName string, addr string, stop chan error) error {
	cli, err := clientv3.New(defaultEtcdConfig)
	if err != nil {
		return fmt.Errorf("create etcd client failed: %v", err)
	}
	defer cli.Close()

	// 创建一个5秒的租约
	resp, err := cli.Grant(context.Background(), 5)
	if err != nil {
		return fmt.Errorf("create lease failed: %v", err)
	}

	// 注册至etcd
	_, err = cli.Put(context.Background(), serviceName+"/"+addr, addr, clientv3.WithLease(resp.ID))
	if err != nil {
		return fmt.Errorf("add etcd record failed: %v", err)
	}

	// 对租约进行续期
	ch, err := cli.KeepAlive(context.Background(), resp.ID)
	if err != nil {
		return fmt.Errorf("set keepalive failed: %v", err)
	}

	log.Printf("[%s] register service ok\n", addr)

	// for循环保证程序不退出，这样就能持续进行续约
	// 循环体内监听三个时间，任何一个时间触发都意味着服务需要结束
	for {
		select {
		case err = <-stop:
			if err != nil {
				log.Println(err)
			}
			return err
		case <-cli.Ctx().Done():
			// 监听etcd客户端的上下文（Context）是否已经被取消或过期
			// 一旦相关的上下文被取消，则结束监听
			log.Println("etcd client service closed")
			return nil
		case _, open := <-ch:
			if !open { // 表明测试心跳的通道关闭
				log.Println("keepalive channel closed")
				_, err = cli.Revoke(context.Background(), resp.ID) // 撤销之前创建的租约
				return err
			}
		}
	}
}

// EtcdDial 向grpc请求一个服务
// 通过提供一个etcd client和service name即可获得Connection
func EtcdDial(c *clientv3.Client, service string) (*grpc.ClientConn, error) {

	resp, err := c.Get(context.Background(), service)
	if err != nil {
		return nil, err
	}

	return grpc.Dial(string(resp.Kvs[0].Value),
		grpc.WithInsecure(),
		grpc.WithBlock(),
	)
}