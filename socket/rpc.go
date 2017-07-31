package socket

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/anacrolix/torrent/iplist"
	"github.com/anacrolix/torrent/util"
	"github.com/mh-cbon/dht/kmsg"
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
		tx:          NewTxServer(c.QueryTimeout),
		id:          c.id,
		ipBlockList: c.ipBlockList,
		mu:          &sync.RWMutex{},
		readOnly:    c.readOnly,
	}
	return ret
}

// QueryHandler handles a query from the remote.
type QueryHandler func(msg kmsg.Msg, remote *net.UDPAddr) error

// RPC is a query/writer of krpc messages with transaction support.
type RPC struct {
	KrpcPacketEncoder
	id             string
	tx             *TxServer
	mu             *sync.RWMutex
	ipBlockList    iplist.Ranger
	OnSendQuery    func(remote *net.UDPAddr, p map[string]interface{})
	OnRcvQuery     func(remote *net.UDPAddr, p kmsg.Msg)
	OnSendResponse func(remote *net.UDPAddr, p map[string]interface{})
	OnRcvResponse  func(remote *net.UDPAddr, p kmsg.Msg)
	readOnly       bool
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

// LogReceiver is an interface to define log callbacks.
type LogReceiver interface {
	OnSendQuery(remote *net.UDPAddr, p map[string]interface{})
	OnRcvQuery(remote *net.UDPAddr, p kmsg.Msg)
	OnSendResponse(remote *net.UDPAddr, p map[string]interface{})
	OnRcvResponse(remote *net.UDPAddr, p kmsg.Msg)
}

// SetLog callbacks.
func (s *RPC) SetLog(l LogReceiver) {
	s.OnSendQuery = l.OnSendQuery
	s.OnRcvQuery = l.OnRcvQuery
	s.OnSendResponse = l.OnSendResponse
	s.OnRcvResponse = l.OnRcvResponse
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
	return s.KrpcPacketEncoder.Listen(func(m kmsg.Msg, addr *net.UDPAddr) (err error) {
		isQ := m.Y == "q"
		// bep43: When a DHT node enters the read-only state,
		// It no longer responds to 'query' messages that it receives,
		// that is messages containing a 'q' flag in the top-level dictionary.
		if isQ && s.readOnly {
			return
		}
		if s.IPBlocked(addr.IP) {
			return fmt.Errorf("IP blocked %v", addr.String())
		}
		if isQ {
			if s.OnRcvQuery != nil {
				s.OnRcvQuery(addr, m)
			}
			if h != nil {
				err = h(m, addr)
			}
			return
		}
		if s.OnRcvResponse != nil {
			s.OnRcvResponse(addr, m)
		}
		_, err = s.tx.HandleResponse(m, addr)
		return
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
	tx, err := s.tx.Prepare(addr, onResponse, func(tx *Tx) error {
		if a == nil {
			a = make(map[string]interface{}, 1)
		}
		a["id"] = s.id
		p := map[string]interface{}{
			"t": tx.id,
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
		if s.OnSendQuery != nil {
			s.OnSendQuery(addr, p)
		}
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
	if s.OnSendResponse != nil {
		s.OnSendResponse(addr, p)
	}
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
	if s.OnSendResponse != nil {
		s.OnSendResponse(addr, p)
	}
	return s.Write(p, addr)
}

// Close is TBD.
func (s *RPC) Close() error {
	s.tx.Stop()
	return s.KrpcPacketEncoder.Close()
}
