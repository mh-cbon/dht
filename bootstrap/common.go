//Package bootstrap provides public common utilities to work with bootstrap addresses
package bootstrap

import (
	"fmt"
	"net"

	"github.com/mh-cbon/dht/bucket"
)

// Public lists public addresses to bootstrap.
var Public = []string{
	"router.utorrent.com:6881",
	"router.bittorrent.com:6881",
}

//Node creates a bootstrap contact, a contact with an address but no ID.
func Node(addr *net.UDPAddr) bucket.ContactIdentifier {
	return bucket.NewContact("", *addr)
}

// Contacts returns contacts for string of network adresses.
func Contacts(nodeAddrs []string) (addrs []bucket.ContactIdentifier, err error) {
	b, err := Addrs(nodeAddrs)
	if err != nil {
		return nil, err
	}
	for _, a := range b {
		addrs = append(addrs, Node(a))
	}
	return addrs, nil
}

// Addrs returns net.UDPAddr slice of given network address strings.
func Addrs(nodeAddrs []string) (addrs []*net.UDPAddr, err error) {
	if len(nodeAddrs) == 0 {
		return nil, fmt.Errorf("nothing to resolve")
	}
	var resolveErr error
	for _, addrStr := range nodeAddrs {
		udpAddr, err2 := net.ResolveUDPAddr("udp4", addrStr)
		if err2 == nil {
			addrs = append(addrs, udpAddr)
		} else {
			resolveErr = err2
		}
	}
	if len(addrs) == 0 {
		if resolveErr != nil {
			err = fmt.Errorf("nothing resolved: %v", resolveErr)
		} else {
			err = fmt.Errorf("nothing resolved: %v", nodeAddrs)
		}
	}
	return
}
