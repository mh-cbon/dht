package rpc

import (
	"net"
	"sync"

	"github.com/mh-cbon/dht/bucket"
)

// TableStore store tables of lookup.
type TableStore struct {
	// lookupTables    *lru.Cache
	tables map[string]*bucket.TSBucket
} //todo: probably add some sort of limit.

// NewStore is a constructor.
func NewStore() *TableStore {
	return &TableStore{
		tables: map[string]*bucket.TSBucket{},
	}
}

// Add a table for given target.
func (t *TableStore) Add(target []byte, table *bucket.TSBucket) bool {
	starget := string(target)
	if _, ok := t.tables[starget]; ok {
		return false
	}
	t.tables[starget] = table
	return true
}

// Get a table for given target.
func (t *TableStore) Get(target []byte) (*bucket.TSBucket, bool) {
	starget := string(target)
	if x, ok := t.tables[starget]; ok {
		return x, ok
	}
	return nil, false
}

// Contains a table with given target.
func (t *TableStore) Contains(target []byte) bool {
	starget := string(target)
	if _, ok := t.tables[starget]; ok {
		return ok
	}
	return false
}

// Rm a table for given target.
func (t *TableStore) Rm(target []byte) bool {
	starget := string(target)
	if _, ok := t.tables[starget]; ok {
		delete(t.tables, starget)
		return ok
	}
	return false
}

// RemoveNode from all tables.
func (t *TableStore) RemoveNode(node *net.UDPAddr) (ret []bucket.ContactIdentifier) {
	for _, table := range t.tables {
		ret = append(ret, table.RemoveByAddr(node)...)
	}
	return ret
}

// ClearAllTables then clear internal storage.
func (t *TableStore) ClearAllTables() {
	for _, table := range t.tables {
		table.Clear()
	}
	t.tables = map[string]*bucket.TSBucket{}
}

// TSTableStore is TS of TableStore.
type TSTableStore struct {
	store *TableStore
	mu    *sync.RWMutex
}

// NewTSStore is a constructor of TS store.
func NewTSStore() *TSTableStore {
	return &TSTableStore{
		store: NewStore(),
		mu:    &sync.RWMutex{},
	}
}

// Add a table for given target.
func (t *TSTableStore) Add(target []byte, table *bucket.TSBucket) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.store.Add(target, table)
}

// Get a table for given target.
func (t *TSTableStore) Get(target []byte) (*bucket.TSBucket, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.store.Get(target)
}

// Contains a table with given target.
func (t *TSTableStore) Contains(target []byte) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.store.Contains(target)
}

// Rm a table for given target.
func (t *TSTableStore) Rm(target []byte) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.store.Rm(target)
}

// RemoveNode from all tables.
func (t *TSTableStore) RemoveNode(node *net.UDPAddr) []bucket.ContactIdentifier {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.store.RemoveNode(node)
}

// ClearAllTables then clear internal storage.
func (t *TSTableStore) ClearAllTables() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.store.ClearAllTables()
}
