package mygroupcache

import (
	pb "my_groupcache/cachepb"
)

// PeerPicker is the interface that must be implemented to locate
// the peer that owns a specific key.
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter is the interface that must be implemented by a peer.
type PeerGetter interface {
	Get(in *pb.Request, out *pb.Response) error
}

var portPicker PeerPicker

func RegisterPeerPicker(p PeerPicker) {
	// if portPicker != nil {
	// 	panic("RegisterPeerPicker called more than once")
	// }
	portPicker = p
}