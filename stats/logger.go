package stats

import (
	"net"

	"github.com/mh-cbon/dht/kmsg"
)

// OnSendQuery logs a query send
func (s *Peers) OnSendQuery(remote *net.UDPAddr, p map[string]interface{}) {}

// OnRcvQuery logs a query reception
func (s *Peers) OnRcvQuery(remote *net.UDPAddr, query kmsg.Msg) {
	addr := remote.String()
	if _, ok := s.stats[addr]; !ok {
		s.stats[addr] = NewPeer(*remote)
	}
	x := s.stats[addr]
	x.OnRcvQuery(remote, query)
	x.Cap(s.maxActivityLength)
}

// OnSendResponse logs a response send
func (s *Peers) OnSendResponse(remote *net.UDPAddr, p map[string]interface{}) {}

// OnSendError logs a error send
func (s *Peers) OnSendError(remote *net.UDPAddr, p map[string]interface{}) {}

// OnRcvResponse logs a response reception
func (s *Peers) OnRcvResponse(remote *net.UDPAddr, queriedQ string, queriedA map[string]interface{}, response kmsg.Msg) {
	addr := remote.String()
	if _, ok := s.stats[addr]; !ok {
		s.stats[addr] = NewPeer(*remote)
	}
	x := s.stats[addr]
	wasTimeout := x.IsTimeout()
	x.OnRcvResponse(remote, queriedQ, queriedA, response)
	x.Cap(s.maxActivityLength)
	if !wasTimeout && x.IsTimeout() {
		for _, f := range s.onPeerTimeout {
			f(remote, queriedQ, queriedA, response)
		}
	}
}

// OnTxNotFound logs a response with a not found transaction (likely to happen if the node has falsely timedout)
func (s *Peers) OnTxNotFound(remote *net.UDPAddr, q kmsg.Msg) {
	addr := remote.String()
	if x, ok := s.stats[addr]; ok {
		x.OnTxNotFound(remote, q)
		x.Cap(s.maxActivityLength)
	}
}

// OnPeerTimeout calls f when a peer goes to timeout.
func (s *Peers) OnPeerTimeout(name string, f PeerTimeout) bool {
	_, ok := s.onPeerTimeout[name]
	if !ok {
		s.onPeerTimeout[name] = f
	}
	return !ok
}

// OffPeerTimeout removes f callback.
func (s *Peers) OffPeerTimeout(name string) bool {
	_, ok := s.onPeerTimeout[name]
	if ok {
		delete(s.onPeerTimeout, name)
	}
	return ok
}
