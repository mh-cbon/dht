package rpc

import (
	"io"
	"net"
	"sync"

	"github.com/mh-cbon/dht/bucket"
	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/socket"
	boom "github.com/tylertreat/BoomFilters"
)

// SocketRPCer is a socket capable of query/answer rpc.
type SocketRPCer interface {
	Listen(h socket.QueryHandler) error
	Close() error
	Query(node *net.UDPAddr, q string, a map[string]interface{}, onResponse func(kmsg.Msg)) (*socket.Tx, error)
	Respond(node *net.UDPAddr, txID string, a kmsg.Return) error
	Error(node *net.UDPAddr, txID string, e kmsg.Error) error
	Addr() *net.UDPAddr
	SetID(id string)
	ID() string
	SetLog(l socket.LogReceiver)
}

// KRPCConfig configures the rpc interface.
type KRPCConfig struct {
	k           int
	concurrency int
}

// WithK nodes par bucket in the tables.
func (c KRPCConfig) WithK(k int) KRPCConfig {
	c.k = k
	return c
}

// WithConcurrency query limit.
func (c KRPCConfig) WithConcurrency(concurrency int) KRPCConfig {
	c.concurrency = concurrency
	return c
}

// Timeout handler.
type Timeout func(q string, a map[string]interface{}, remote *net.UDPAddr, e kmsg.Error)

// KRPC is rpc on kadmelia table.
type KRPC struct {
	config               KRPCConfig
	socket               SocketRPCer
	mu                   *sync.RWMutex
	onNodeTimeout        Timeout
	badNodes             *boom.BloomFilter // badNodes to be cleaned from the lookup tables, used by bep43 to ban ro nodes
	bootstrap            *bucket.TSBucket  // our location in the network we are connected to.
	lookupTableForPeers  *TSTableStore     // bep05
	lookupTableForStores *TSTableStore     // bep44
	peerStatsLogger      PeerStatLogger
}

// New Kadmelia rpc of socket.
func New(s SocketRPCer, c KRPCConfig) *KRPC {
	if c.concurrency < 1 {
		c.concurrency = 3
	}
	if c.k < 1 {
		c.k = 20
	}
	ret := &KRPC{
		config:               c,
		socket:               s,
		mu:                   &sync.RWMutex{},
		badNodes:             boom.NewBloomFilter(3500, 0.5),
		lookupTableForPeers:  NewTSStore(),
		lookupTableForStores: NewTSStore(),
		peerStatsLogger:      NewTSPeerStatsLogger(NewPeerStatsLogger()),
	}
	ret.OnTimeout(nil) // force create a callback to cleanup lookup tables.
	return ret
}

// SetLog callbacks.
func (k *KRPC) SetLog(l socket.LogReceiver) {
	k.socket.SetLog(l)
}

// OnTimeout registers a callback called when a node timeout.
func (k *KRPC) OnTimeout(t Timeout) *KRPC {
	k.onNodeTimeout = func(q string, a map[string]interface{}, node *net.UDPAddr, err kmsg.Error) {
		k.lookupTableForPeers.RemoveNode(node)
		k.lookupTableForStores.RemoveNode(node)
		if t != nil {
			t(q, a, node, err)
		}
	}
	return k
}

// Close the socket.
func (k *KRPC) Close() error {
	k.onNodeTimeout = nil
	k.badNodes.Reset()
	if k.bootstrap != nil {
		k.bootstrap.Clear()
	}
	k.lookupTableForPeers.ClearAllTables()
	k.lookupTableForStores.ClearAllTables()
	return k.socket.Close()
}

// Listen the socket.
func (k *KRPC) Listen(h socket.QueryHandler) error {
	h = SecuredQueryOnly(k, BanRoNodes(k, h))
	return k.socket.Listen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
		go k.peerStatsLogger.OnRcvQuery(remote, msg)
		return h(msg, remote)
	})
}

// MustListen the socket might panic.
func (k *KRPC) MustListen(h socket.QueryHandler) {
	err := k.Listen(h)
	if err != nil && err != io.EOF {
		panic(err)
	}
}

// Addr returns socket Addr.
func (k *KRPC) Addr() *net.UDPAddr {
	return k.socket.Addr()
}

// ID  returns your node id.
func (k *KRPC) ID() string {
	return k.socket.ID()
}

// PeerStatLogger is a kind of logger to establish peer statistics.
type PeerStatLogger interface {
	OnSendQuery(remote *net.UDPAddr, q string, a map[string]interface{})
	OnRcvQuery(remote *net.UDPAddr, query kmsg.Msg)
	OnSendResponse(remote *net.UDPAddr, txID string, a kmsg.Return)
	OnSendError(remote *net.UDPAddr, txID string, e kmsg.Error)
	OnRcvResponse(remote *net.UDPAddr, fromQ string, fromA map[string]interface{}, response kmsg.Msg)
	IsTimeout(remote *net.UDPAddr) bool
	IsActive(remote *net.UDPAddr) bool
	GoodNodes(nodes []kmsg.NodeInfo) []bucket.ContactIdentifier
	IsBadNode(addr *net.UDPAddr) bool
	AddBadNode(addr *net.UDPAddr)
}

// SetPeerStatsLogger peer stats logger.
func (k *KRPC) SetPeerStatsLogger(l PeerStatLogger) {
	k.peerStatsLogger = l
}

// Query a node.
func (k *KRPC) Query(node *net.UDPAddr, q string, a map[string]interface{}, onResponse func(kmsg.Msg)) (*socket.Tx, error) {
	go k.peerStatsLogger.OnSendQuery(node, q, a)
	return k.socket.Query(node, q, a, func(res kmsg.Msg) {
		if res.E != nil && k.onNodeTimeout != nil {
			if res.E.Code == 201 {
				go k.onNodeTimeout(q, a, node, *res.E)
			}
		}
		go k.peerStatsLogger.OnRcvResponse(node, q, a, res)
		if onResponse != nil {
			onResponse(res)
		}
	})
}

// Respond to node.
func (k *KRPC) Respond(node *net.UDPAddr, txID string, a kmsg.Return) error {
	go k.peerStatsLogger.OnSendResponse(node, txID, a)
	return k.socket.Respond(node, txID, a)
}

// Error respond an error to node.
func (k *KRPC) Error(node *net.UDPAddr, txID string, e kmsg.Error) error {
	go k.peerStatsLogger.OnSendError(node, txID, e)
	return k.socket.Error(node, txID, e)
}

// VisitIndex is the func sgnature to visit slices.
type VisitIndex func(int, chan<- error) (*socket.Tx, error)

//Batch queries, calls for visit.
func (k *KRPC) Batch(n int, visit VisitIndex) []error {
	var ret []error
	done := make(chan error, n)
	go func() {
		for i := 0; i < n; i++ {
			e := i
			go func() {
				if _, err := visit(e, done); err != nil {
					done <- err
				}
			}()
		}
	}()
	for i := 0; i < n; i++ {
		e := <-done
		if y, ok := e.(*kmsg.Error); ok {
			if y != nil {
				ret = append(ret, e)
			}
		} else if e != nil {
			ret = append(ret, e)
		}
	}
	close(done)
	return ret
}

// VisitAddr is the func sgnature to visit addresses.
type VisitAddr func(*net.UDPAddr, chan<- error) (*socket.Tx, error)

//BatchAddrs queries, calls for visit.
func (k *KRPC) BatchAddrs(addrs []*net.UDPAddr, visit VisitAddr) []error {
	return k.Batch(len(addrs), func(i int, done chan<- error) (*socket.Tx, error) {
		return visit(addrs[i], done)
	})
}

// VisitNode is the func sgnature to visit nodes.
type VisitNode func(bucket.ContactIdentifier, chan<- error) (*socket.Tx, error)

//BatchNodes queries, calls for visit.
func (k *KRPC) BatchNodes(nodes []bucket.ContactIdentifier, visit VisitNode) []error {
	return k.Batch(len(nodes), func(i int, done chan<- error) (*socket.Tx, error) {
		return visit(nodes[i], done)
	})
}
