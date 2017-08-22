package dht

import (
	"net"

	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/socket"
)

// StdQueryHandler is the standard query handler for this implementation.
func StdQueryHandler(d *DHT) socket.QueryHandler {
	return func(msg kmsg.Msg, remote *net.UDPAddr) error {
		q := msg.Q

		if q == kmsg.QFindNode {
			return d.OnFindNode(msg, remote)

		} else if q == kmsg.QPing {
			return d.OnPing(msg, remote)

		} else if q == kmsg.QAnnouncePeer {
			return d.OnAnnouncePeer(msg, remote)

		} else if q == kmsg.QGetPeers {
			return d.OnGetPeers(msg, remote)

		} else if q == kmsg.QGet {
			return d.OnGet(msg, remote)

		} else if q == kmsg.QPut {
			return d.OnPut(msg, remote)
		}

		return d.Error(remote, msg.T, kmsg.ErrorMethodUnknown)
	}
}
