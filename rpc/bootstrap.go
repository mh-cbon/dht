package rpc

import (
	"fmt"
	"sync"

	"github.com/anacrolix/torrent/util"
	"github.com/mh-cbon/dht/bootstrap"
	"github.com/mh-cbon/dht/bucket"
	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/socket"
	"github.com/mh-cbon/dht/stats"
	boom "github.com/tylertreat/BoomFilters"
)

type emptyBootstrap struct {
	error
}

func (e emptyBootstrap) IsEmptyBootstrap() bool { return true }

type ipRecommendation struct {
	Count int
	p     *util.CompactPeer
}

// Boostrap your participation to the network.
// It returns a table results of the bootstrap.
// bep42: If a recommanded ip is returned, you should consider saving that ip as your public ip,
// derive a new id, and restart the bootstrap.
// if an error is returned, the table should not be considere as correctly built.
func (k *KRPC) Boostrap(target []byte, publicIP *util.CompactPeer, addrs []string) (*util.CompactPeer, error) {

	if target == nil {
		target = make([]byte, 20)
	}
	k.socket.SetID(string(target))
	if k.bootstrap != nil {
		k.bootstrap.Clear()
	}
	k.bootstrap = k.MakeTable(target)

	table := k.bootstrap
	if publicIP == nil {
		k := k.socket.Addr()
		publicIP = &util.CompactPeer{IP: k.IP, Port: k.Port}
	}

	var recommendedIP *util.CompactPeer
	ipRecommendations := map[string]*ipRecommendation{}
	var knownPublicIP string
	if publicIP != nil {
		knownPublicIP = fmt.Sprintf("%v", publicIP.IP)
	}
	var notRecommanded int
	mu := &sync.RWMutex{}
	recommendIP := func(res kmsg.Msg) {
		if res.IP.IP != nil {
			mu.Lock()
			defer mu.Unlock()
			ipstring := fmt.Sprintf("%v", res.IP.IP)
			if ipstring != knownPublicIP {
				if x, ok := ipRecommendations[ipstring]; ok {
					x.Count++
				} else {
					ipRecommendations[ipstring] = &ipRecommendation{p: &res.IP}
				}
			} else {
				notRecommanded++
			}
		}
	}

	bnContacts, err := bootstrap.Contacts(addrs)
	if err != nil {
		if len(addrs) == 0 {
			err = emptyBootstrap{err}
		}
		return nil, err
	}

	queriedNodes := boom.NewBloomFilter(1500, 0.5)
	var ncontacts []bucket.ContactIdentifier
	var gotErr []error
	for i := 0; i < len(bnContacts); i += 8 {
		u := k.config.concurrency
		if u > i+len(bnContacts) {
			u = len(bnContacts)
		}
		todoBNodes := bnContacts[i:u]

		errs := k.BatchNodes(todoBNodes, func(contact bucket.ContactIdentifier, done chan<- error) (*socket.Tx, error) {
			return k.FindNode(contact.GetAddr(), target, func(res kmsg.Msg) {
				if res.E != nil {
					table.RemoveByAddr(contact.GetAddr())
				} else if res.Y == "r" && res.R != nil {
					recommendIP(res) // bep42
					for _, node := range k.goodNodes(res.R.Nodes) {
						table.Add(node)
					}
				}
				done <- res.E
			})
		})
		gotErr = append(gotErr, errs...)
		ncontacts = table.Closest(target, 8)
		if len(ncontacts) == 8 {
			break
		}
	}

	if len(ncontacts) == 0 {
		if len(gotErr) > 0 {
			return nil, fmt.Errorf("All bootstrap nodes failed %v", gotErr)
		}
		return nil, fmt.Errorf("The table is empty after bootstrap")
	}

	queriedNodes.Reset()
	hops := 0

	for {
		// using the bootstrapped table, search for closest nodes to us until contacts.len is zero.
		// contacts len reduce if it contains already queried nodes.
		contacts := k.notQueried(queriedNodes, ncontacts)
		if len(contacts) == 0 {
			break
		}
		hops++
		k.BatchNodes(contacts, func(contact bucket.ContactIdentifier, done chan<- error) (*socket.Tx, error) {
			return k.FindNode(contact.GetAddr(), target, func(res kmsg.Msg) {
				if res.E != nil {
					table.RemoveByAddr(contact.GetAddr())
				} else if res.Y == "r" && res.R != nil && res.SenderID() == string(contact.GetID()) {
					recommendIP(res) // bep42
					for _, node := range k.goodNodes(res.R.Nodes) {
						table.Add(node)
					}
				}
				done <- res.E
			})
		})
		ncontacts = table.Closest(target, 8)
	}

	// bep42: A DHT node which receives an ip result in a request SHOULD consider restarting its DHT node with a new node ID,
	// taking this IP into account.
	// Since a single node can not be trusted,
	// there should be some mechanism to determine
	// whether or not the node has a correct understanding
	// of its external IP or not.
	// This could be done by voting, or only restart the DHT once at least a certain number of nodes,
	// from separate searches, tells you your node ID is incorrect.
	if len(ipRecommendations) > 0 {
		c := 0
		for _, v := range ipRecommendations {
			if v.Count > c {
				c = v.Count
				if c > notRecommanded {
					recommendedIP = v.p
				}
			}
		}
	}

	return recommendedIP, err
}

func (k *KRPC) notQueried(queriedNodes *boom.BloomFilter, contacts []bucket.ContactIdentifier) []bucket.ContactIdentifier {
	if contacts == nil {
		return []bucket.ContactIdentifier{}
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	ret := []bucket.ContactIdentifier{}
	for _, c := range contacts {
		if !queriedNodes.TestAndAdd([]byte(c.GetAddr().String())) {
			ret = append(ret, c)
		}
	}
	return ret
}

func (k *KRPC) goodNodes(nodes []kmsg.NodeInfo) (ret []bucket.ContactIdentifier) {
	if nodes == nil {
		return ret
	}
	k.GetPeersStats().Transact(func(p *stats.Peers) {
		for _, n := range nodes {
			if !p.IsBanned(n.Addr) &&
				!p.IsTimeout(n.Addr) &&
				!p.IsRO(n.Addr) &&
				p.LastIDValid(n.Addr) {
				ret = append(ret, NewNode(n.ID, n.Addr)) //todo: consider detect nodes being queried.
			}
		}
	})
	return ret
}

// handleTablePing handles the ping event emitted on the table,
// it is very standard thing about removing useless nodes, adding new ones if positions are available.
func (k *KRPC) handleTablePing(table *bucket.TSBucket) bucket.PingFunc {
	return func(oldContacts []bucket.ContactIdentifier, newishContact bucket.ContactIdentifier) {
		todoContacts := []bucket.ContactIdentifier{}
		k.GetPeersStats().Transact(func(p *stats.Peers) {
			for _, o := range oldContacts {
				if p.IsActive(o.GetAddr()) && !p.IsTimeout(o.GetAddr()) {
					todoContacts = append(todoContacts, o)
				}
			}
		})
		errs := k.BatchNodes(todoContacts, func(c bucket.ContactIdentifier, done chan<- error) (*socket.Tx, error) {
			return k.Ping(c.GetAddr(), func(msg kmsg.Msg) {
				if msg.E != nil {
					table.Remove(c.GetID())
				} else {
					table.Add(c)
				}
				done <- msg.E
			})
		})
		if len(errs) > 0 {
			table.Add(newishContact)
		}
	}
}

// BootstrapExport the nodes in the bootstrap table.
func (k *KRPC) BootstrapExport() (ret []string) {
	for _, c := range k.bootstrap.ToArray() {
		ret = append(ret, c.GetAddr().String())
	}
	return
}
