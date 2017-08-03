package dht

import (
	"net"
	"time"

	"github.com/anacrolix/torrent/iplist"
	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/logger"
	"github.com/mh-cbon/dht/rpc"
	"github.com/mh-cbon/dht/socket"
	"github.com/mh-cbon/dht/token"
)

//Opt is a option setter
type Opt func(*DHT)

//ToKRPC ...
func ToKRPC(in rpc.KRPCOpt) Opt {
	return func(d *DHT) {
		in(d.rpc)
	}
}

// Opts are dht server options.
var Opts = struct {
	WithRPCSocket   func(r rpc.SocketRPCer) Opt
	WithSocket      func(socket net.PacketConn) Opt
	WithAddr        func(addr string) Opt
	WithTimeout     func(duraton time.Duration) Opt
	ReadOnly        func(ro bool) Opt
	ID              func(id string) Opt
	BlockIPs        func(ips iplist.Ranger) Opt
	WithK           func(k int) Opt
	WithConcurrency func(concurrency int) Opt
	WithTokenSecret func(secret []byte) Opt
}{
	WithRPCSocket: func(r rpc.SocketRPCer) Opt {
		return ToKRPC(rpc.KRPCOpts.WithRPCSocket(r))
	},
	WithSocket: func(socket net.PacketConn) Opt {
		return ToKRPC(rpc.KRPCOpts.WithSocket(socket))
	},
	WithAddr: func(addr string) Opt {
		return ToKRPC(rpc.KRPCOpts.WithAddr(addr))
	},
	WithTimeout: func(duration time.Duration) Opt {
		return ToKRPC(rpc.KRPCOpts.WithTimeout(duration))
	},
	ReadOnly: func(ro bool) Opt {
		return ToKRPC(rpc.KRPCOpts.ReadOnly(ro))
	},
	ID: func(id string) Opt {
		return ToKRPC(rpc.KRPCOpts.ID(id))
	},
	BlockIPs: func(ips iplist.Ranger) Opt {
		return ToKRPC(rpc.KRPCOpts.BlockIPs(ips))
	},
	WithK: func(k int) Opt {
		return ToKRPC(rpc.KRPCOpts.WithK(k))
	},
	WithConcurrency: func(concurrency int) Opt {
		return ToKRPC(rpc.KRPCOpts.WithConcurrency(concurrency))
	},
	WithTokenSecret: func(secret []byte) Opt {
		return func(d *DHT) {
			d.tokenServer.SetSecret(secret)
		}
	},
}

// DefaultOps for a dht node
var DefaultOps = func() Opt {
	return func(d *DHT) {
		Opts.WithRPCSocket(socket.NewConcurrent(24))(d)
		Opts.WithTimeout(time.Second)(d)
		Opts.WithK(20)(d)
		Opts.WithConcurrency(8)(d)
	}
}

// DHT implements the bep
type DHT struct {
	rpc *rpc.KRPC // kademlia/dht protocol impl. of a transactionnal udp query/response socket.

	// validates incoming token query
	tokenServer *token.Server // bep05 - bep44

	// stores tokens provided by remotes
	bep05TokenStore *token.TSStore // bep05
	// stores peer announces
	peerStore *TSPeerStore // bep05

	// stores tokens provided by remotes
	bep44TokenStore *token.TSStore // bep44
	// stores put request
	bep44ValueStore *TSValueStore // bep44
}

// New initiliazes a new DHT.
func New(opts ...Opt) *DHT {
	ret := &DHT{
		rpc:             rpc.New(),
		tokenServer:     token.NewDefault(nil),
		peerStore:       NewTSPeerStore(),
		bep05TokenStore: token.NewTSStore(),
		bep44TokenStore: token.NewTSStore(),
		bep44ValueStore: NewTSValueStore(),
	}
	for _, opt := range opts {
		opt(ret)
	}
	ret.rpc.GetPeersStats().OnPeerTimeout("dht.dht", ret.RmNodeFromStores)
	return ret
}

// RmNodeFromStores removes given node from stores.
func (d *DHT) RmNodeFromStores(remote *net.UDPAddr, queriedQ string, queriedA map[string]interface{}, response kmsg.Msg) {
	d.peerStore.RemPeer(Peer{IP: remote.IP, Port: remote.Port})
	d.bep05TokenStore.RmByAddr(remote)
	d.bep44TokenStore.RmByAddr(remote)
}

// ListenAndServe the socket, handle aueries with given handler, calls for ready if listen is ok.
func (d *DHT) ListenAndServe(h socket.QueryHandler, ready func(*DHT) error) error {
	e := make(chan error)
	go func() {
		e <- d.Listen(h)
	}()
	select {
	case err := <-e:
		return err
	case <-time.After(time.Millisecond * 10):
	}
	return ready(d)
}

// Listen the socket, handle aueries with given handler.
func (d *DHT) Listen(h socket.QueryHandler) error {
	return d.rpc.Listen(h)
}

// Serve the socket and execute ready func if the listen operation succeeded.
func (d *DHT) Serve(ready func(*DHT) error) error {
	return d.ListenAndServe(StdQueryHandler(d), ready)
}

// GetAddr returns the local address.
func (d *DHT) GetAddr() *net.UDPAddr {
	return d.rpc.GetAddr()
}

// ID  returns your node id (raw string).
func (d *DHT) ID() string {
	return d.rpc.ID()
}

// GetID  returns your node id (raw byte string).
func (d *DHT) GetID() []byte {
	return d.rpc.GetID()
}

// AddLogger of this rpc.
func (d *DHT) AddLogger(l logger.LogReceiver) {
	d.rpc.AddLogger(l)
}

// RmLogger of this rpc.
func (d *DHT) RmLogger(l logger.LogReceiver) bool {
	return d.rpc.RmLogger(l)
}

// Close the dht and its socket.
func (d *DHT) Close() error {
	d.rpc.GetPeersStats().OffPeerTimeout("dht.dht")
	d.peerStore.Clear()
	d.bep05TokenStore.Clear()
	d.bep44TokenStore.Clear()
	d.bep44ValueStore.Clear()
	return d.rpc.Close()
}

// ValidateToken validates a token received from the network queries.
func (d *DHT) ValidateToken(token string, remote *net.UDPAddr) bool {
	return d.tokenServer.ValidToken(token, remote)
}

// TokenGetPeers returns a token saved after sending get_peers queries.
func (d *DHT) TokenGetPeers(remote net.UDPAddr) string {
	return d.bep05TokenStore.GetToken(remote)
}

// TokenGet returns a token saved after sending get queries.
func (d *DHT) TokenGet(remote net.UDPAddr) string {
	return d.bep44TokenStore.GetToken(remote)
}

// Query an addr.
func (d *DHT) Query(addr *net.UDPAddr, q string, a map[string]interface{}, onResponse func(kmsg.Msg)) (*socket.Tx, error) {
	return d.rpc.Query(addr, q, a, onResponse)
}

// Respond to node.
func (d *DHT) Respond(node *net.UDPAddr, txID string, a kmsg.Return) error {
	return d.rpc.Respond(node, txID, a)
}

// Error respond an error to node.
func (d *DHT) Error(node *net.UDPAddr, txID string, e kmsg.Error) error {
	return d.rpc.Error(node, txID, e)
}
