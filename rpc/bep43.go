package rpc

import (
	"net"

	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/socket"
)

// BanRoNodes marks ro nodes as bad nodes.
// bep43: A node that receives DHT messages should inspect incoming queries for the 'ro' flag set to 1.
// If it is found, the node should not add the message sender to its routing table.
func BanRoNodes(k *KRPC, f socket.QueryHandler) socket.QueryHandler {
	return func(msg kmsg.Msg, remote *net.UDPAddr) error {
		if msg.RO == 1 {
			// todo: ideally there a mechanism to instantly remove the node from known tables.
			k.addBadNode(remote)
		}
		return f(msg, remote)
	}
}

func (k *KRPC) addBadNode(addr *net.UDPAddr) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.badNodes.Add([]byte(addr.String()))
	return
}

func (k *KRPC) isBadNode(addr *net.UDPAddr) bool {
	return k.badNodes.Test([]byte(addr.String()))
}
