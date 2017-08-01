package rpc

import (
	"net"
	"sync"
	"time"

	"github.com/mh-cbon/dht/bucket"
	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/security"
)

func NewPeerStatsLogger() *PeerStats {
	return &PeerStats{
		stats:             map[string]*PeerStat{},
		maxActivityLength: 10,
		activeDuration:    time.Second * 30,
	}
}

// PeerStats gather and maintain stats about peers.
type PeerStats struct {
	maxActivityLength int
	activeDuration    time.Duration
	stats             map[string]*PeerStat
	badNodes          []string
}

// OnSendQuery logs a query send
func (p *PeerStats) OnSendQuery(remote *net.UDPAddr, q string, a map[string]interface{}) {}

// OnRcvQuery logs a query reception
func (p *PeerStats) OnRcvQuery(remote *net.UDPAddr, query kmsg.Msg) {
	addr := remote.String()
	if _, ok := p.stats[addr]; !ok {
		p.stats[addr] = &PeerStat{*remote, []*PeerActivity{}}
	}

	p.stats[addr].prependActivity(p.maxActivityLength, PeerActivity{kind: "q", id: query.A.ID, ro: query.RO == 1, date: time.Now()})
}

// OnSendResponse logs a response send
func (p *PeerStats) OnSendResponse(remote *net.UDPAddr, txID string, a kmsg.Return) {}

// OnSendError logs a error send
func (p *PeerStats) OnSendError(remote *net.UDPAddr, txID string, e kmsg.Error) {}

// OnRcvResponse logs a response reception
func (p *PeerStats) OnRcvResponse(remote *net.UDPAddr, fromQ string, fromA map[string]interface{}, response kmsg.Msg) {
	addr := remote.String()
	if _, ok := p.stats[addr]; !ok {
		p.stats[addr] = &PeerStat{*remote, []*PeerActivity{}}
	}
	var a PeerActivity
	if response.E != nil {
		a = PeerActivity{kind: "e", err: *response.E, date: time.Now()}
	} else {
		a = PeerActivity{kind: "r", id: response.R.ID, date: time.Now()}
	}
	p.stats[addr].prependActivity(p.maxActivityLength, a)
}

// IsTimeout return true if remote has timedout
func (p *PeerStats) IsTimeout(remote *net.UDPAddr) bool {
	addr := remote.String()
	if x, ok := p.stats[addr]; ok {
		return x.IsTimeout()
	}
	return false
}

// IsActive return true if remote has good query since duration
func (p *PeerStats) IsActive(remote *net.UDPAddr) bool {
	addr := remote.String()
	if x, ok := p.stats[addr]; ok {
		return x.IsActive(p.activeDuration)
	}
	return false
}

// IsRO return true if remote has sent query with ro flag
func (p *PeerStats) IsRO(remote *net.UDPAddr) bool {
	addr := remote.String()
	if x, ok := p.stats[addr]; !ok {
		return x.IsRO()
	}
	return false
}

func (p *PeerStats) AddBadNode(addr *net.UDPAddr) {
	p.badNodes = append(p.badNodes, addr.String())
	// k.mu.Lock()
	// defer k.mu.Unlock()
	// k.badNodes.Add([]byte(addr.String()))
	// return
}

func (p *PeerStats) IsBadNode(addr *net.UDPAddr) bool {
	r := addr.String()
	for _, a := range p.badNodes {
		if a == r {
			return true
		}
	}
	if x, ok := p.stats[r]; ok {
		return (x.IsTimeout() || x.IsRO() || !x.LastIDValid())
	}
	return false
}

func (p *PeerStats) GoodNodes(nodes []kmsg.NodeInfo) []bucket.ContactIdentifier {
	ret := []bucket.ContactIdentifier{}
	if nodes == nil {
		return ret
	}
	for _, n := range nodes {
		if p.IsBadNode(n.Addr) == false {
			ret = append(ret, NewNode(n.ID, n.Addr)) //todo: consider detect nodes being queried.
		}
	}
	return ret
}

func NewTSPeerStatsLogger(store *PeerStats) *TSPeerStats {
	return &TSPeerStats{store: store, mu: &sync.RWMutex{}}
}

// TSPeerStats is a TS peer stats store.
type TSPeerStats struct {
	store *PeerStats
	mu    *sync.RWMutex
}

func (t *TSPeerStats) OnSendQuery(remote *net.UDPAddr, q string, a map[string]interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.store.OnSendQuery(remote, q, a)
}
func (t *TSPeerStats) OnRcvQuery(remote *net.UDPAddr, query kmsg.Msg) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.store.OnRcvQuery(remote, query)
}
func (t *TSPeerStats) OnSendResponse(remote *net.UDPAddr, txID string, a kmsg.Return) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.store.OnSendResponse(remote, txID, a)
}
func (t *TSPeerStats) OnSendError(remote *net.UDPAddr, txID string, e kmsg.Error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.store.OnSendError(remote, txID, e)
}
func (t *TSPeerStats) OnRcvResponse(remote *net.UDPAddr, fromQ string, fromA map[string]interface{}, response kmsg.Msg) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.store.OnRcvResponse(remote, fromQ, fromA, response)
}
func (t *TSPeerStats) IsTimeout(remote *net.UDPAddr) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.store.IsTimeout(remote)
}
func (t *TSPeerStats) IsActive(remote *net.UDPAddr) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.store.IsActive(remote)
}
func (t *TSPeerStats) GoodNodes(nodes []kmsg.NodeInfo) []bucket.ContactIdentifier {
	ret := []bucket.ContactIdentifier{}
	if nodes == nil {
		return ret
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.store.GoodNodes(nodes)
}

func (t *TSPeerStats) AddBadNode(addr *net.UDPAddr) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.store.AddBadNode(addr)
}

func (t *TSPeerStats) IsBadNode(addr *net.UDPAddr) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.store.IsBadNode(addr)
}

type PeerStat struct {
	addr     net.UDPAddr
	activity []*PeerActivity
}

func (p *PeerStat) prependActivity(max int, a PeerActivity) {
	act := p.activity
	// max := 20
	if len(act) < max {
		act = append([]*PeerActivity{&a}, act...)
	} else {
		for i := 0; i < max-1; i++ {
			act[i] = act[i+1]
		}
		act[0] = &a
	}
	p.activity = act
}

func (p *PeerStat) IsTimeout() bool {
	i := 0
	for _, a := range p.activity {
		if a.kind == "e" && a.err.Code == kmsg.ErrorTimeout.Code {
			i++
		} else {
			break
		}
	}
	return i > 2
}
func (p *PeerStat) IsActive(d time.Duration) bool {
	for _, a := range p.activity {
		if a.kind == "e" && a.err.Code == kmsg.ErrorTimeout.Code {
			return false
		} else {
			return a.date.After(time.Now().Add(d))
		}
	}
	return false
}
func (p *PeerStat) IsRO() bool {
	for _, a := range p.activity {
		if a.kind == "q" {
			return a.ro
		}
	}
	return false
}
func (p *PeerStat) LastIDValid() bool {
	id := p.LastID()
	if id == "" {
		return true
	}
	return security.NodeIDSecure(id, p.addr.IP)
}
func (p *PeerStat) LastID() string {
	for _, a := range p.activity {
		if a.kind == "q" || a.kind == "r" {
			return a.id
		}
	}
	return ""
}

type PeerActivity struct {
	kind string
	id   string // receive query
	ro   bool   // receive query
	err  kmsg.Error
	date time.Time
}
