package token

import (
	"fmt"
	"net"
	"sync"
)

// Store of tokens received after sending "get_peers"/"get" query.
type Store struct {
	tokens map[string]string
} //todos: add a limit on maximum number of tokens.

// NewStore creates a new Store
func NewStore() *Store {
	return &Store{tokens: map[string]string{}}
}

//SetToken for given remote
func (s *Store) SetToken(token string, remote *net.UDPAddr) error {
	id := remote.String()
	s.tokens[id] = token
	return nil
}

//RmByAddr for given remote.
func (s *Store) RmByAddr(remote *net.UDPAddr) error {
	id := remote.String()
	if _, ok := s.tokens[id]; ok {
		delete(s.tokens, id)
		return nil
	}
	return fmt.Errorf("remote address not found: %v", id)
}

//RmByToken for given token
func (s *Store) RmByToken(token string) error {
	for id, t := range s.tokens {
		if t == token {
			delete(s.tokens, id)
		}
	}
	return nil
}

//GetToken for a remote.
func (s *Store) GetToken(remote net.UDPAddr) string {
	id := remote.String()
	if token, ok := s.tokens[id]; ok {
		return token
	}
	return ""
}

//Clear the storage.
func (s *Store) Clear() {
	s.tokens = map[string]string{}
}

// TSStore tokens for a given address
type TSStore struct {
	store *Store
	mu    *sync.RWMutex
}

// NewTSStore creates a new TS store
func NewTSStore() *TSStore {
	return &TSStore{
		store: NewStore(),
		mu:    &sync.RWMutex{},
	}
}

//SetToken for given remote
func (s *TSStore) SetToken(token string, remote *net.UDPAddr) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.store.SetToken(token, remote)
}

//RmByAddr for given remote.
func (s *TSStore) RmByAddr(remote *net.UDPAddr) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.store.RmByAddr(remote)
}

//RmByToken for given token
func (s *TSStore) RmByToken(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.store.RmByToken(token)
}

//GetToken for a remote.
func (s *TSStore) GetToken(remote net.UDPAddr) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.store.GetToken(remote)
}

//Clear the storage.
func (s *TSStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store.Clear()
}
