// Package stats is a logger that maintains information about nodes and their quality given observed traffic.
package stats

import (
	"net"
	"time"

	"github.com/mh-cbon/dht/kmsg"
)

// PeerTimeout is the signature of the callback invoked when a nodes enters timeout.
type PeerTimeout func(remote *net.UDPAddr, queriedQ string, queriedA map[string]interface{}, response kmsg.Msg)

// NewPeersLogger with default options (maxActivityLength:10, activeDuration:30s)
func NewPeersLogger() *Peers {
	return &Peers{
		stats:             map[string]*Peer{},
		maxActivityLength: 10,
		activeDuration:    time.Second * 30,
		onPeerTimeout:     map[string]PeerTimeout{},
	}
}

// Peers is a logger to maintain peers quality information.
type Peers struct {
	maxActivityLength int
	activeDuration    time.Duration
	stats             map[string]*Peer
	banNodes          []string
	onPeerTimeout     map[string]PeerTimeout
}

// IsTimeout return true if remote has timedout (at least three times in a row), false if never queried.
func (s *Peers) IsTimeout(remote *net.UDPAddr) bool {
	addr := remote.String()
	if x, ok := s.stats[addr]; ok {
		return x.IsTimeout()
	}
	return false
}

// IsActive return true if remote has good query within duration, false if never queried.
func (s *Peers) IsActive(remote *net.UDPAddr) bool {
	addr := remote.String()
	if x, ok := s.stats[addr]; ok {
		return x.IsActive(s.activeDuration)
	}
	return false
}

// LastIDValid return false if the last query/response of remote contained an invalid id, true if never queried.
func (s *Peers) LastIDValid(remote *net.UDPAddr) bool {
	addr := remote.String()
	if x, ok := s.stats[addr]; ok {
		return x.LastIDValid()
	}
	return true
}

// IsRO return true if the remote has sent a query with an ro flag, false if never queried.
func (s *Peers) IsRO(remote *net.UDPAddr) bool {
	addr := remote.String()
	if x, ok := s.stats[addr]; ok {
		return x.IsRO()
	}
	return false
}

// BanNode permanently whatsoever.
func (s *Peers) BanNode(addr *net.UDPAddr) {
	s.banNodes = append(s.banNodes, addr.String())
}

// Unban previously banned node.
func (s *Peers) Unban(addr *net.UDPAddr) {
	bans := []string{}
	a := addr.String()
	for _, h := range s.banNodes {
		if h != a {
			bans = append(bans, h)
		}
	}
	s.banNodes = bans
}

// IsBanned returns true if the node is permanently banned.
func (s *Peers) IsBanned(addr *net.UDPAddr) bool {
	r := addr.String()
	for _, a := range s.banNodes {
		if a == r {
			return true
		}
	}
	return false
}

// Clear the storage.
func (s *Peers) Clear() {
	s.stats = map[string]*Peer{}
	s.banNodes = s.banNodes[:0]
	s.onPeerTimeout = map[string]PeerTimeout{}
}
