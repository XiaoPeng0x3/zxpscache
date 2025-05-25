package registry

import (
	"context"
	"fmt"
	"testing"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func TestRegisterServiceToETCD(t *testing.T) {
	stop := make(chan error)
	erro := fmt.Errorf("stop")

	go func() {
		err := RegisterServiceToETCD("Hello", "127.0.0.1:8089", stop)
		if err != nil && err != erro {
			t.Error(err)
		}
	}()
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Errorf("create etcd client failed: %v", err)
	}
	defer cli.Close()

	time.Sleep(time.Second * 3)
	resp, err := cli.Get(context.Background(), "Hello/", clientv3.WithPrefix())

	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	for _, ev := range resp.Kvs {
		t.Logf("%s:%s\n", ev.Key, ev.Value)
		if string(ev.Key) != "Hello/127.0.0.1:8089" || string(ev.Value) != "127.0.0.1:8089" {
			t.Error("key value not we want")
		}
	}

	time.Sleep(time.Second * 2)
	stop <- erro
}