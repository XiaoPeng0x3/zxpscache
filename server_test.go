package mygroupcache

import (
	"testing"

)

func Test_PeerRelation(t *testing.T) {
	addrMap := map[string]string{
		"8001": "127.0.0.1:8001",
		"8002": "127.0.0.1:8002",
		"8003": "127.0.0.1:8003",
	}
	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}
	s := NewGRPCPool("127.0.0.1:8001", 0, nil)
	s.SetPeers(addrs...)

	peer, ok := s.PickPeer("tom")
	if !ok {
		t.Errorf("PickPeer error")
	}
	if peer.(*client).name != "groupcache/127.0.0.1:8003" {
		t.Errorf("pick wrong peer")
	}
}