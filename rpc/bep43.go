package rpc

import (
	"net"

	"github.com/mh-cbon/dht/stats"
)

//todo: refactor, its not in the right file anymore.

// BanNode permanently whatsoever.
func (k *KRPC) BanNode(addr *net.UDPAddr) {
	k.GetPeersStats().BanNode(addr)
}

// IsBanned if the node was previously banned.
func (k *KRPC) IsBanned(addr *net.UDPAddr) bool {
	return k.GetPeersStats().IsBanned(addr)
}

// IsBadNode if the node is banned, has timeout, is ro, or last id is invalid.
func (k *KRPC) IsBadNode(addr *net.UDPAddr) bool {
	ret := false
	k.GetPeersStats().Transact(func(p *stats.Peers) {
		if p.IsBanned(addr) ||
			p.IsTimeout(addr) ||
			p.IsRO(addr) ||
			p.LastIDValid(addr) == false {
			ret = true
		}
	})
	return ret
}
