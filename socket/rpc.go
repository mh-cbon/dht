package socket

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/anacrolix/torrent/iplist"
	"github.com/anacrolix/torrent/util"
	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/logger"
	"github.com/mh-cbon/dht/stats"
)

// KrpcPacketEncoder reads/writes kmsg.Message.
type KrpcPacketEncoder interface {
	Listen(read func(kmsg.Msg, *net.UDPAddr) error) error
	Write(m map[string]interface{}, addr *net.UDPAddr) error
	Close() error
	Addr() *net.UDPAddr
}

// RPCConfig configures the RPC socket.
type RPCConfig struct {
	ServerConfig
	QueryTimeout time.Duration // duration before a query is considered as timeout.
	id           string
	ipBlockList  iplist.Ranger
	readOnly     bool
}

// WithTimeout on query response.
func (c RPCConfig) WithTimeout(q time.Duration) RPCConfig {
	c.QueryTimeout = q
	return c
}

// WithID of the socket.
func (c RPCConfig) WithID(id []byte) RPCConfig {
	c.id = string(id)
	return c
}

// WithAddr of the socket.
func (c RPCConfig) WithAddr(addr string) RPCConfig {
	c.ServerConfig = c.ServerConfig.WithAddr(addr)
	return c
}

// WithIPBlockList configures the sanitizer to exclude ip matching given ranger
func (c RPCConfig) WithIPBlockList(i iplist.Ranger) RPCConfig {
	c.ipBlockList = i
	return c
}

// ReadOnly node.
func (c RPCConfig) ReadOnly(readOnly bool) RPCConfig {
	c.readOnly = readOnly
	return c
}

// NewConfig prepares a default configuration.
func NewConfig(addr string) RPCConfig {
	return RPCConfig{}.WithAddr(addr)
}

// New Server with given config.
func New(c RPCConfig) *RPC {
	ret := &RPC{
		KrpcPacketEncoder: &Bencoded{
			Server: NewServer(c.ServerConfig),
		},
		tx:              NewTxServer(c.QueryTimeout),
		id:              c.id,
		ipBlockList:     c.ipBlockList,
		mu:              &sync.RWMutex{},
		readOnly:        c.readOnly,
		peerStatsLogger: stats.NewTSPeersLogger(stats.NewPeersLogger()),
		logReceiver:     logger.NewMany(),
	}
	ret.logReceiver.Add(ret.peerStatsLogger)
	return ret
}

// QueryHandler handles a query from the remote.
type QueryHandler func(msg kmsg.Msg, remote *net.UDPAddr) error

// RPC is a query/writer of krpc messages with transaction support.
type RPC struct {
	KrpcPacketEncoder
	id              string
	tx              *TxServer
	mu              *sync.RWMutex
	ipBlockList     iplist.Ranger
	peerStatsLogger *stats.TSPeers
	logReceiver     *logger.Many
	readOnly        bool
}

// GetAddr implements bucket.ContactIdentifier.
func (s *RPC) GetAddr() *net.UDPAddr {
	return s.Addr()
}

// GetID implements bucket.ContactIdentifier.
func (s *RPC) GetID() []byte {
	return []byte(s.id)
}

// SetID of your node.
func (s *RPC) SetID(id string) {
	s.id = id
}

// ID of your node.
func (s *RPC) ID() string {
	return s.id
}

// ReadOnly node, true/false.
func (s *RPC) ReadOnly(ro bool) {
	s.readOnly = ro
}

// GetPeersStats of this rpc.
func (s *RPC) GetPeersStats() *stats.TSPeers {
	return s.peerStatsLogger
}

// AddLogger of this rpc.
func (s *RPC) AddLogger(l logger.LogReceiver) {
	s.logReceiver.Add(l)
}

// RmLogger of this rpc.
func (s *RPC) RmLogger(l logger.LogReceiver) bool {
	return s.logReceiver.Rm(l)
}

// IPBlocked returns true when the node matches ipBlockList.
func (s *RPC) IPBlocked(ip net.IP) (blocked bool) {
	if s.ipBlockList == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, blocked = s.ipBlockList.Lookup(ip)
	return
}

// Listen reads kmsg.Msg
func (s *RPC) Listen(h QueryHandler) error {
	queryHandler := func(msg kmsg.Msg, remote *net.UDPAddr) error {
		s.logReceiver.OnRcvQuery(remote, msg)
		if h != nil {
			return h(msg, remote)
		}
		return nil
	}
	return s.KrpcPacketEncoder.Listen(func(m kmsg.Msg, addr *net.UDPAddr) error {
		isQ := m.Y == "q"
		// bep43: When a DHT node enters the read-only state,
		// It no longer responds to 'query' messages that it receives,
		// that is messages containing a 'q' flag in the top-level dictionary.
		if isQ && s.readOnly {
			return nil
		}
		if s.IPBlocked(addr.IP) {
			return fmt.Errorf("IP blocked %v", addr.String())
		}
		if isQ {
			return queryHandler(m, addr)
		}
		tx, err := s.tx.HandleResponse(m, addr)
		if err != nil {
			log.Println(tx)
			s.logReceiver.OnTxNotFound(addr, m)
		}
		return err
	})
}

// MustListen panics if the server fails to listen.
func (s *RPC) MustListen(h QueryHandler) *RPC {
	if err := s.Listen(h); err != nil && err != io.EOF {
		panic(err)
	}
	return s
}

// Query given addr with given query and message arguments.
func (s *RPC) Query(addr *net.UDPAddr, q string, a map[string]interface{}, onResponse func(kmsg.Msg)) (*Tx, error) {
	if s.IPBlocked(addr.IP) {
		return nil, fmt.Errorf("IP blocked %v", addr.String())
	}
	responseHandler := func(res kmsg.Msg) {
		s.logReceiver.OnRcvResponse(addr, q, a, res)
		if onResponse != nil {
			onResponse(res)
		}
	}
	tx, err := s.tx.Prepare(addr, responseHandler, func(tx *Tx) error {
		if a == nil {
			a = make(map[string]interface{}, 1)
		}
		a["id"] = s.id
		p := map[string]interface{}{
			"t": tx.txID,
			"y": "q",
			"q": q,
			"a": a,
		}
		// bep43: When a DHT node enters the read-only state
		// In each outgoing query message the read-only DHT node
		// places a 'ro' key in the top-level message dictionary
		// and sets its value to 1.
		// This will appear in the request as '2:roi1e'.
		if s.readOnly {
			p["ro"] = 1
		}
		s.logReceiver.OnSendQuery(addr, p)
		return s.KrpcPacketEncoder.Write(p, addr)
	})
	return tx, err
}

// Respond to given address with given transaction id and given Return argument.
// bep42: all DHT responses SHOULD include a top-level field called ip,
// containing a compact binary representation of the requestor's IP and port.
// That is big endian IP followed by 2 bytes of big endian port.
func (s *RPC) Respond(addr *net.UDPAddr, txID string, r kmsg.Return) error {
	if s.IPBlocked(addr.IP) {
		return fmt.Errorf("IP blocked %v", addr.String())
	}
	// r.ID = string(s.BID())
	r.ID = s.id
	p := map[string]interface{}{
		"t":  txID,
		"y":  "r",
		"r":  r,
		"ip": util.CompactPeer{IP: addr.IP.To4(), Port: addr.Port},
	}
	s.logReceiver.OnSendResponse(addr, p)
	return s.Write(p, addr)
}

// Error sends given error to given address.
// bep42: all DHT responses SHOULD include a top-level field called ip,
// containing a compact binary representation of the requestor's IP and port.
// That is big endian IP followed by 2 bytes of big endian port.
func (s *RPC) Error(addr *net.UDPAddr, txID string, e kmsg.Error) error {
	if s.IPBlocked(addr.IP) {
		return fmt.Errorf("IP blocked %v", addr.String())
	}
	p := map[string]interface{}{
		"t":  txID,
		"y":  "e",
		"e":  e,
		"ip": util.CompactPeer{IP: addr.IP.To4(), Port: addr.Port},
	}
	s.logReceiver.OnSendResponse(addr, p)
	return s.Write(p, addr)
}

// Close is TBD.
func (s *RPC) Close() error {
	// s.peerStatsLogger.Clear()
	s.logReceiver.Clear()
	s.tx.Stop()
	return s.KrpcPacketEncoder.Close()
}
