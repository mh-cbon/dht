package rpc

import (
	"net"

	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/socket"
)

// QFindNode is the "find_node" query verb.
var QFindNode = "find_node"

// FindNode sends a "find_node" query.
func (k *KRPC) FindNode(addr *net.UDPAddr, target []byte, onResponse func(kmsg.Msg)) (*socket.Tx, error) {
	a := map[string]interface{}{"target": target}
	return k.Query(addr, QFindNode, a, onResponse)
}

// QPing is the "ping" query verb.
var QPing = "ping"

// Ping sends a "ping" query.
func (k *KRPC) Ping(addr *net.UDPAddr, onResponse func(kmsg.Msg)) (*socket.Tx, error) {
	return k.Query(addr, QPing, nil, onResponse)
}

// QGetPeers is the "get_peers" query verb.
var QGetPeers = "get_peers"

// GetPeers sends a "get_peers" query.
func (k *KRPC) GetPeers(addr *net.UDPAddr, target []byte, onResponse func(kmsg.Msg)) (*socket.Tx, error) {
	a := map[string]interface{}{"info_hash": target}
	// bep42: guards against bad node id
	return k.Query(addr, QGetPeers, a, SecuredResponseOnly(addr, onResponse))
}

// QAnnouncePeer is the "announce_peer" query verb.
var QAnnouncePeer = "announce_peer"

// AnnouncePeer sends a "announce_peer" query.
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

	return k.Query(addr, QAnnouncePeer, a, onResponse)
}
