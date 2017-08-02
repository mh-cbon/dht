package stats

import (
	"net"
	"sync"

	"github.com/mh-cbon/dht/kmsg"
)

// NewTSPeersLogger is a TS Peers
func NewTSPeersLogger(store *Peers) *TSPeers {
	return &TSPeers{store: store, mu: &sync.RWMutex{}}
}

// TSPeers is a TS peer stats store.
type TSPeers struct {
	store *Peers
	mu    *sync.RWMutex
}

// Transact is tx
func (t *TSPeers) Transact(f func(p *Peers)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	f(t.store)
}

// BanNode permanently whatsoever.
func (t *TSPeers) BanNode(addr *net.UDPAddr) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.store.BanNode(addr)
}

// Unban node previously banned node.
func (t *TSPeers) Unban(addr *net.UDPAddr) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.store.Unban(addr)
}

// IsBanned returns true if the node is permanently banned.
func (t *TSPeers) IsBanned(addr *net.UDPAddr) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.store.IsBanned(addr)
}

// Clear the storage.
func (t *TSPeers) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.store.Clear()
}

// OnSendQuery logs the sending of a query.
func (t *TSPeers) OnSendQuery(remote *net.UDPAddr, p map[string]interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.store.OnSendQuery(remote, p)
}

// OnRcvQuery logs the reception of a query.
func (t *TSPeers) OnRcvQuery(remote *net.UDPAddr, query kmsg.Msg) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.store.OnRcvQuery(remote, query)
}

// OnSendResponse logs the sending of a response.
func (t *TSPeers) OnSendResponse(remote *net.UDPAddr, p map[string]interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.store.OnSendResponse(remote, p)
}

// OnSendError logs the sending of an error.
func (t *TSPeers) OnSendError(remote *net.UDPAddr, p map[string]interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.store.OnSendError(remote, p)
}

// OnRcvResponse logs a response received after a query.
func (t *TSPeers) OnRcvResponse(remote *net.UDPAddr, fromQ string, fromA map[string]interface{}, response kmsg.Msg) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.store.OnRcvResponse(remote, fromQ, fromA, response)
}

// OnTxNotFound logs a response with a not found transaction (likely to happen if the node has falsely timedout)
func (t *TSPeers) OnTxNotFound(remote *net.UDPAddr, p kmsg.Msg) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.store.OnTxNotFound(remote, p)
}

// OnPeerTimeout calls f when a peer goes to timeout.
func (t *TSPeers) OnPeerTimeout(name string, f PeerTimeout) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.store.OnPeerTimeout(name, f)
}

// OffPeerTimeout removes f callback.
func (t *TSPeers) OffPeerTimeout(name string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.store.OffPeerTimeout(name)
}
