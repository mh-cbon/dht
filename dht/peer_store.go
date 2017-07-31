package dht

import (
	"net"
	"strconv"
	"sync"
)

// Peer is a node announcing a torrent
type Peer struct {
	IP   net.IP
	Port int
}

func (p *Peer) String() string {
	return net.JoinHostPort(p.IP.String(), strconv.FormatInt(int64(p.Port), 10))
}

// PeerStore provides peers for an announce, or announces for peer.
type PeerStore struct {
	peers map[string][]Peer
}

// NewPeerStore initialize a store of announce->peers
func NewPeerStore() *PeerStore {
	return &PeerStore{
		peers: map[string][]Peer{},
	}
}

// AddPeer for given announceTarget
func (s *PeerStore) AddPeer(announceTarget string, p Peer) bool {
	if peers, ok := s.peers[announceTarget]; ok {
		peers = append(peers, p)
		s.peers[announceTarget] = peers
	} else {
		s.peers[announceTarget] = []Peer{p}
	}
	return true
}

// Get peers for given announceTarget
func (s *PeerStore) Get(announceTarget string) []Peer {
	if peers, ok := s.peers[announceTarget]; ok {
		return peers
	}
	return []Peer{}
}

// RemPeerAnnounce with given announceTarget
func (s *PeerStore) RemPeerAnnounce(announceTarget string, p Peer) bool {
	if peers, ok := s.peers[announceTarget]; ok {
		index := -1
		for i, peer := range peers {
			if peer.String() == p.String() {
				index = i
				break
			}
		}
		if index > -1 {
			peers = append(peers[:index], peers[index:]...)
			s.peers[announceTarget] = peers
			return true
		}
	}
	return false
}

// RemPeer with given address
func (s *PeerStore) RemPeer(p Peer) bool {
	for a := range s.peers {
		s.RemPeerAnnounce(a, p)
	}
	return false
}

//RemAnnounce and all its peers.
func (s *PeerStore) RemAnnounce(announceTarget string) bool {
	delete(s.peers, announceTarget)
	return true
}

//Clear the storage.
func (s *PeerStore) Clear() {
	s.peers = map[string][]Peer{}
}

// TSPeerStore is a TS PeerStore.
type TSPeerStore struct {
	store *PeerStore
	mu    *sync.RWMutex
}

// NewTSPeerStore is a TS PeerStore
func NewTSPeerStore() *TSPeerStore {
	return &TSPeerStore{
		store: NewPeerStore(),
		mu:    &sync.RWMutex{},
	}
}

// AddPeer for given announceTarget
func (s *TSPeerStore) AddPeer(announceTarget string, p Peer) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.store.AddPeer(announceTarget, p)
}

// Get peers for given announceTarget
func (s *TSPeerStore) Get(announceTarget string) []Peer {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.store.Get(announceTarget)
}

// RemPeer with given announceTarget
func (s *TSPeerStore) RemPeer(p Peer) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.store.RemPeer(p)
}

// RemPeerAnnounce with given announceTarget
func (s *TSPeerStore) RemPeerAnnounce(announceTarget string, p Peer) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.store.RemPeerAnnounce(announceTarget, p)
}

//RemAnnounce and all its peers.
func (s *TSPeerStore) RemAnnounce(announceTarget string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.store.RemAnnounce(announceTarget)
}

//Clear the storage.
func (s *TSPeerStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store.Clear()
}
