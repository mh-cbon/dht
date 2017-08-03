package rpc

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/anacrolix/torrent/iplist"
	"github.com/mh-cbon/dht/bucket"
	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/logger"
	"github.com/mh-cbon/dht/socket"
	"github.com/mh-cbon/dht/stats"
)

// SocketRPCer is a socket capable of query/answer rpc.
type SocketRPCer interface {
	socket.RPCConfigurer
	Listen(h socket.QueryHandler) error
	Close() error
	Query(node *net.UDPAddr, q string, a map[string]interface{}, onResponse func(kmsg.Msg)) (*socket.Tx, error)
	Respond(node *net.UDPAddr, txID string, a kmsg.Return) error
	Error(node *net.UDPAddr, txID string, e kmsg.Error) error
	GetAddr() *net.UDPAddr
	GetID() []byte
	ID() string
	GetPeersStats() *stats.TSPeers
	AddLogger(l logger.LogReceiver)
	RmLogger(l logger.LogReceiver) bool
}

//ToRPC ...
func ToRPC(in socket.RPCOpt) KRPCOpt {
	return func(s *KRPC) {
		in(s.rpc.(socket.RPCConfigurer))
	}
}

//KRPCOpt is a option setter
type KRPCOpt func(*KRPC)

// KRPCOpts are krpc options.
var KRPCOpts = struct {
	WithRPCSocket   func(r SocketRPCer) KRPCOpt
	WithSocket      func(socket net.PacketConn) KRPCOpt
	WithAddr        func(addr string) KRPCOpt
	WithTimeout     func(duraton time.Duration) KRPCOpt
	ReadOnly        func(ro bool) KRPCOpt
	ID              func(id string) KRPCOpt
	BlockIPs        func(ips iplist.Ranger) KRPCOpt
	WithK           func(k int) KRPCOpt
	WithConcurrency func(concurrency int) KRPCOpt
}{
	WithTimeout: func(duration time.Duration) KRPCOpt {
		return ToRPC(socket.RPCOpts.WithTimeout(duration))
	},
	WithSocket: func(pc net.PacketConn) KRPCOpt {
		return ToRPC(socket.RPCOpts.WithSocket(pc))
	},
	WithAddr: func(addr string) KRPCOpt {
		return ToRPC(socket.RPCOpts.WithAddr(addr))
	},
	ReadOnly: func(ro bool) KRPCOpt {
		return ToRPC(socket.RPCOpts.ReadOnly(ro))
	},
	ID: func(id string) KRPCOpt {
		return ToRPC(socket.RPCOpts.ID(id))
	},
	BlockIPs: func(ips iplist.Ranger) KRPCOpt {
		return ToRPC(socket.RPCOpts.BlockIPs(ips))
	},
	WithRPCSocket: func(r SocketRPCer) KRPCOpt {
		return func(s *KRPC) {
			s.rpc = r
		}
	},
	WithK: func(k int) KRPCOpt {
		return func(s *KRPC) {
			s.k = k
		}
	},
	WithConcurrency: func(concurrency int) KRPCOpt {
		return func(s *KRPC) {
			s.concurrency = concurrency
		}
	},
}

// Timeout handler.
type Timeout func(q string, a map[string]interface{}, remote *net.UDPAddr, e kmsg.Error)

// KRPC is rpc on kadmelia table.
type KRPC struct {
	k                    int
	concurrency          int
	rpc                  SocketRPCer
	mu                   *sync.RWMutex
	bootstrap            *bucket.TSBucket // our location in the network we are connected to.
	lookupTableForPeers  *TSTableStore    // bep05
	lookupTableForStores *TSTableStore    // bep44
}

// New Kadmelia rpc of socket.
func New(opts ...KRPCOpt) *KRPC {
	ret := &KRPC{
		k:                    20,
		concurrency:          3,
		rpc:                  socket.New(),
		mu:                   &sync.RWMutex{},
		lookupTableForPeers:  NewTSStore(),
		lookupTableForStores: NewTSStore(),
	}
	for _, opt := range opts {
		opt(ret)
	}
	if !ret.rpc.GetPeersStats().OnPeerTimeout("dht.rpc", ret.RmNodeFromLookupTables) {
		panic("nop not good, fix that")
	}
	return ret
}

// GetPeersStats of this rpc.
func (k *KRPC) GetPeersStats() *stats.TSPeers {
	return k.rpc.GetPeersStats()
}

// AddLogger of this rpc.
func (k *KRPC) AddLogger(l logger.LogReceiver) {
	k.rpc.AddLogger(l)
}

// RmLogger of this rpc.
func (k *KRPC) RmLogger(l logger.LogReceiver) bool {
	return k.rpc.RmLogger(l)
}

// RmNodeFromLookupTables removes given node from lookup table.
func (k *KRPC) RmNodeFromLookupTables(remote *net.UDPAddr, queriedQ string, queriedA map[string]interface{}, response kmsg.Msg) {
	k.lookupTableForPeers.RemoveNode(remote)
	k.lookupTableForStores.RemoveNode(remote)
}

// Close the socket.
func (k *KRPC) Close() error {
	k.GetPeersStats().OffPeerTimeout("dht.rpc")
	if k.bootstrap != nil {
		k.bootstrap.Clear()
	}
	k.lookupTableForPeers.ClearAllTables()
	k.lookupTableForStores.ClearAllTables()
	return k.rpc.Close()
}

// Listen the socket.
func (k *KRPC) Listen(h socket.QueryHandler) error {
	return k.rpc.Listen(SecuredQueryOnly(k, h))
}

// MustListen the socket might panic.
func (k *KRPC) MustListen(h socket.QueryHandler) {
	err := k.Listen(h)
	if err != nil && err != io.EOF {
		panic(err)
	}
}

// GetAddr returns socket Addr.
func (k *KRPC) GetAddr() *net.UDPAddr {
	return k.rpc.GetAddr()
}

// GetID implements bucket.ContactIdentifier (raw string).
func (k *KRPC) GetID() []byte {
	return k.rpc.GetID()
}

// ID  returns your node id (raw string).
func (k *KRPC) ID() string {
	return k.rpc.ID()
}

// Query a node.
func (k *KRPC) Query(node *net.UDPAddr, q string, a map[string]interface{}, onResponse func(kmsg.Msg)) (*socket.Tx, error) {
	return k.rpc.Query(node, q, a, func(res kmsg.Msg) {
		if onResponse != nil {
			onResponse(res)
		}
	})
}

// Respond to node.
func (k *KRPC) Respond(node *net.UDPAddr, txID string, a kmsg.Return) error {
	return k.rpc.Respond(node, txID, a)
}

// Error respond an error to node.
func (k *KRPC) Error(node *net.UDPAddr, txID string, e kmsg.Error) error {
	return k.rpc.Error(node, txID, e)
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
