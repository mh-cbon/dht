package rpc

import (
	"net"

	"github.com/mh-cbon/dht/bucket"
	"github.com/mh-cbon/dht/kmsg"
)

//Node is a contact on the network.
type Node struct {
	bucket.ContactIdentifier
}

//NewNode creates a new contact.
func NewNode(id [20]byte, addr *net.UDPAddr) Node {
	return Node{
		ContactIdentifier: bucket.NewContact(string(id[:]), *addr),
	}
}

// NodeInfo produces kmsg.NodeInfo from a contact.
func NodeInfo(c bucket.ContactIdentifier) kmsg.NodeInfo {
	i := [20]byte{}
	id := c.GetID()
	for index := range id {
		if index < len(i) {
			i[index] = id[index]
		}
	}
	return kmsg.NodeInfo{ID: i, Addr: c.GetAddr()}
}
