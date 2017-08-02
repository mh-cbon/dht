package dht

import (
	"crypto/rand"
	"net"
	"time"

	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/logger"
	"github.com/mh-cbon/dht/rpc"
	"github.com/mh-cbon/dht/socket"
	"github.com/mh-cbon/dht/token"
)

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
// secret is the token server secret.
// r is a kademlia/dht api to interact with the network.
func New(secret []byte, r *rpc.KRPC) *DHT {
	if secret == nil {
		secret = make([]byte, 20)
		rand.Read(secret)
	}
	if len(secret) > 20 {
		secret = secret[:20]
	}
	if r == nil {
		sockConfig := socket.RPCConfig{}.WithTimeout(time.Second)
		// socket := socket.New(sockConfig)
		socket := socket.NewConcurrent(24, sockConfig)

		kconfig := rpc.KRPCConfig{}.WithConcurrency(8).WithK(20)
		r = rpc.New(socket, kconfig)
	}
	ret := &DHT{
		rpc:             r,
		tokenServer:     token.NewDefault(secret),
		peerStore:       NewTSPeerStore(),
		bep05TokenStore: token.NewTSStore(),
		bep44TokenStore: token.NewTSStore(),
		bep44ValueStore: NewTSValueStore(),
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

// Listen to the socket and execute given func if the listen operation succeeded.
func (d *DHT) Listen(ready func(*DHT) error) error {
	e := make(chan error)
	go func() {
		e <- d.rpc.Listen(d.handleIncomingQuery)
	}()
	select {
	case err := <-e:
		return err
	case <-time.After(time.Millisecond * 10):
	}
	return ready(d)
}

// Addr returns the local address.
func (d *DHT) Addr() *net.UDPAddr {
	return d.rpc.Addr()
}

// ID  returns your node id.
func (d *DHT) ID() string {
	return d.rpc.ID()
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
	// d.tokenServer.Clear()
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

// handleQuery from the network.
func (d *DHT) handleIncomingQuery(msg kmsg.Msg, remote *net.UDPAddr) error {
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
	return d.rpc.Error(remote, msg.T, kmsg.ErrorMethodUnknown)
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
