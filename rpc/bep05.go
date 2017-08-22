package rpc

import (
	"net"

	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/socket"
)

// FindNode send a "find_node" query.
func (k *KRPC) FindNode(addr *net.UDPAddr, target []byte, onResponse func(kmsg.Msg)) (*socket.Tx, error) {
	a := map[string]interface{}{"target": target}
	return k.Query(addr, kmsg.QFindNode, a, onResponse)
}

// Ping send a "ping" query.
func (k *KRPC) Ping(addr *net.UDPAddr, onResponse func(kmsg.Msg)) (*socket.Tx, error) {
	return k.Query(addr, kmsg.QPing, nil, onResponse)
}

// GetPeers send a "get_peers" query.
func (k *KRPC) GetPeers(addr *net.UDPAddr, target []byte, onResponse func(kmsg.Msg)) (*socket.Tx, error) {
	a := map[string]interface{}{"info_hash": target}
	// bep42: guards against bad node id
	return k.Query(addr, kmsg.QGetPeers, a, SecuredResponseOnly(addr, onResponse))
}

// AnnouncePeer send a "announce_peer" query.
func (k *KRPC) AnnouncePeer(addr *net.UDPAddr, target []byte, writeToken string, port uint, impliedPort bool, onResponse func(kmsg.Msg)) (*socket.Tx, error) {
	i := func(impliedPort bool) int {
		if impliedPort {
			return 1
		}
		return 0
	}(impliedPort)
	a := map[string]interface{}{
		"info_hash":    target,
		"token":        writeToken,
		"port":         port,
		"implied_port": i,
	}

	return k.Query(addr, kmsg.QAnnouncePeer, a, onResponse)
}
