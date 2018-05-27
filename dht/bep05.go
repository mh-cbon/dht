package dht

import (
	"encoding/hex"
	"fmt"
	"net"

	"github.com/mh-cbon/dht/util"
	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/rpc"
	"github.com/mh-cbon/dht/socket"
)

// OnPing respond to a ping query.
func (d *DHT) OnPing(msg kmsg.Msg, remote *net.UDPAddr) error {
	return d.Respond(remote, msg.T, kmsg.Return{})
}

// Ping send a ping query to given address.
func (d *DHT) Ping(addr *net.UDPAddr, onResponse func(kmsg.Msg)) (*socket.Tx, error) {
	return d.rpc.Ping(addr, onResponse)
}

// PingAll given addresses.
func (d *DHT) PingAll(addrs []*net.UDPAddr, onResponse func(*net.UDPAddr, kmsg.Msg)) []error {
	return d.rpc.BatchAddrs(addrs, func(remote *net.UDPAddr, done chan<- error) (*socket.Tx, error) {
		return d.Ping(remote, func(res kmsg.Msg) {
			if onResponse != nil {
				onResponse(remote, res)
			}
			done <- res.E
		})
	})
}

// OnFindNode respond to a find_node query.
// When a node receives a find_node query, it should respond with a key "nodes"
// and value of a string containing the compact node info for the target node
// or the K (8) closest good nodes in its own routing table.
func (d *DHT) OnFindNode(msg kmsg.Msg, remote *net.UDPAddr) error {
	if msg.A == nil {
		return fmt.Errorf("bad message no A: %v", msg)
	}

	target := []byte(msg.A.Target)
	if len(target) != 20 {
		return fmt.Errorf("bad target len: %v", msg)
	}

	// todo: complete the implementation to
	// return a key nodes and value of a string containing the compact node (contact information) info for the target node.

	var nodes kmsg.CompactIPv4NodeInfo
	contacts, err := d.rpc.ClosestLocation(target, 8)
	if err == nil && len(contacts) > 0 {
		nodes = []kmsg.NodeInfo{}
		for _, c := range contacts {
			nodes = append(nodes, rpc.NodeInfo(c))
		}
	}

	return d.Respond(remote, msg.T, kmsg.Return{
		Nodes: nodes,
	})
}

// FindNode send a find_node query to given address.
// find_node finds the contact information for a node ID.
// hexTarget is the node's id we're looking for.
func (d *DHT) FindNode(addr *net.UDPAddr, hexTarget string, onResponse func(kmsg.Msg)) (*socket.Tx, error) {
	target, e := hex.DecodeString(hexTarget)
	if e != nil {
		return nil, e
	}
	return d.rpc.FindNode(addr, target, onResponse)
}

// FindNodeAll send find_node msg to all addresses.
func (d *DHT) FindNodeAll(addrs []*net.UDPAddr, hexTarget string, onResponse func(*net.UDPAddr, kmsg.Msg)) []error {
	return d.rpc.BatchAddrs(addrs, func(remote *net.UDPAddr, done chan<- error) (*socket.Tx, error) {
		return d.FindNode(remote, hexTarget, func(res kmsg.Msg) {
			if onResponse != nil {
				onResponse(remote, res)
			}
			done <- res.E
		})
	})
}

// OnGetPeers respond to a get_peers query.
// If the queried node has peers for the infohash, they are returned in a key "values" as a list of strings.
// Each string containing "compact" format peer information for a single peer.
// If the queried node has no peers for the infohash,
// a key "nodes" is returned containing the K nodes in the queried nodes routing table closest to the infohash supplied in the query.
// In either case a "token" key is also included in the return value.
// The token value is a required argument for a future announce_peer query.
// The token value is a short binary string.
func (d *DHT) OnGetPeers(msg kmsg.Msg, remote *net.UDPAddr) error {
	if msg.A == nil {
		return fmt.Errorf("bad message no A: %v", msg)
	}

	target := msg.A.InfoHash
	if len(target) != 20 {
		return fmt.Errorf("bad info_hash len: %v", msg)
	}

	var nodes kmsg.CompactIPv4NodeInfo
	var values []util.CompactPeer

	peers := d.peerStore.Get(target)
	if len(peers) > 0 {
		if len(peers) > 8 {
			peers = peers[:8]
		}
		v := []util.CompactPeer{}
		for _, p := range peers {
			v = append(v, util.CompactPeer{IP: p.IP, Port: p.Port})
		}
		values = v

	} else {
		// lookup for closests nodes to given target in the bootstrap table.
		// A possible optimization is to search into queries tables (get_peers/get)
		// to select the closest one for given target
		// and use this to select the 8th closest nodes.
		// this optimization should provide closer results to the querier.
		contacts, err := d.rpc.ClosestLocation([]byte(target), 8)
		if err == nil && len(contacts) > 0 {
			nodes = []kmsg.NodeInfo{}
			for _, c := range contacts {
				nodes = append(nodes, rpc.NodeInfo(c))
			}
		}
	}
	return d.Respond(remote, msg.T, kmsg.Return{
		Nodes:  nodes,
		Values: values,
		Token:  d.tokenServer.CreateToken(remote),
	})
}

// GetPeers send a get_peers query to given address.
// Get peers associated with a torrent infohash.
// hexTarget is the info hash of the torrent we are looking for.
func (d *DHT) GetPeers(addr *net.UDPAddr, hexTarget string, onResponse func(kmsg.Msg)) (*socket.Tx, error) {
	target, e := hex.DecodeString(hexTarget)
	if e != nil {
		return nil, e
	}
	return d.rpc.GetPeers(addr, target, func(res kmsg.Msg) {
		if res.E == nil && res.R != nil {
			d.bep05TokenStore.SetToken(res.R.Token, addr)
		}
		if onResponse != nil {
			onResponse(res)
		}
	})
}

// GetPeersAll send get_peers message to all addresses.
func (d *DHT) GetPeersAll(addrs []*net.UDPAddr, hexTarget string, onResponse func(*net.UDPAddr, kmsg.Msg)) []error {
	return d.rpc.BatchAddrs(addrs, func(remote *net.UDPAddr, done chan<- error) (*socket.Tx, error) {
		return d.GetPeers(remote, hexTarget, func(res kmsg.Msg) {
			if onResponse != nil {
				onResponse(remote, res)
			}
			done <- res.E
		})
	})
}

// OnAnnouncePeer respond to an announce_peer query.
// The queried node must verify that the token was previously sent to the same IP address as the querying node.
// Then the queried node should store the IP address of the querying node
// and the supplied port number under the infohash in its store of peer contact information.
func (d *DHT) OnAnnouncePeer(msg kmsg.Msg, remote *net.UDPAddr) error {
	if msg.A == nil {
		return fmt.Errorf("bad message no A: %v", msg)
	}

	target := msg.A.InfoHash
	if len(target) != 20 {
		return fmt.Errorf("bad info_hash len: %v", msg)
	}

	token := msg.A.Token
	if token == "" {
		return fmt.Errorf("bad token len: %v", token)
	}

	if d.tokenServer.ValidToken(token, remote) == false {
		return d.Error(remote, msg.T, kmsg.ErrorBadToken)
	}

	peer := &Peer{IP: remote.IP, Port: msg.A.Port}
	if msg.A.ImpliedPort != 0 {
		peer.Port = remote.Port
	}
	added := d.peerStore.AddPeer(target, *peer)
	if added == false {
		return fmt.Errorf("add peer failed added: %v", added)
	}

	return d.Respond(remote, msg.T, kmsg.Return{})
}

// AnnouncePeer send an announce_peer query to given address.
// Announce that the peer, controlling the querying node, is downloading a torrent on a port.
// targetID is 20-byte infohash of target torrent.
// port is the port number the node can be contacted.
// impliedPort is a bool to indicate that remote nodes considers the UDP port rather than announced port. Put to 1|true if you want to use UTP.
func (d *DHT) AnnouncePeer(addr *net.UDPAddr, hexTarget string, port uint, impliedPort bool, onResponse func(kmsg.Msg)) (*socket.Tx, error) {
	target, e := hex.DecodeString(hexTarget)
	if e != nil {
		return nil, e
	}
	writeToken := d.bep05TokenStore.GetToken(*addr)
	if writeToken == "" {
		return d.GetPeers(addr, hexTarget, func(res kmsg.Msg) {
			writeToken = res.R.Token
			if res.E == nil && res.R != nil && writeToken != "" {
				d.rpc.AnnouncePeer(addr, target, writeToken, port, impliedPort, onResponse)
			} else {
				onResponse(kmsg.Msg{})
			}
		})
	}
	return d.rpc.AnnouncePeer(addr, target, writeToken, port, impliedPort, onResponse)
}

// AnnouncePeerAll send an announce_peer query to all addr.
func (d *DHT) AnnouncePeerAll(addrs []*net.UDPAddr, hexTarget string, port uint, impliedPort bool, onResponse func(*net.UDPAddr, kmsg.Msg)) []error {
	return d.rpc.BatchAddrs(addrs, func(remote *net.UDPAddr, done chan<- error) (*socket.Tx, error) {
		return d.AnnouncePeer(remote, hexTarget, port, impliedPort, func(res kmsg.Msg) {
			if onResponse != nil {
				onResponse(remote, res)
			}
			done <- res.E
		})
	})
}
