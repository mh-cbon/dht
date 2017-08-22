package dht

import (
	"fmt"
	"sync"
	"time"
)

// StoredValue is a value stored following a put request.
type StoredValue struct {
	Value        string
	Seq          int
	Cas          int
	K            []byte
	Sig          []byte
	CreationDate time.Time
}

// HasExpired returns true if the value is expired.
func (s StoredValue) HasExpired(d time.Duration) bool {
	return time.Now().After(s.CreationDate.Add(d))
}

// Touch pushes back the expiration by updating its CreationDate.
func (s StoredValue) Touch() {
	s.CreationDate = time.Now()
}

func newStoredValue(v string) *StoredValue {
	return &StoredValue{CreationDate: time.Now(), Value: v}
}

// ValueStore store values of put requests.
type ValueStore struct {
	values map[string]*StoredValue
} //todo: add a kind of limit on number of keys / or max storage size.

// NewValueStore is a constructor.
func NewValueStore() *ValueStore {
	return &ValueStore{
		values: map[string]*StoredValue{},
	}
}

// Add a value for given target, it returns an error if the value already exist.
func (t *ValueStore) Add(target string, value string) error {
	if _, ok := t.values[target]; !ok {
		t.values[target] = newStoredValue(value)
		return nil
	}
	return fmt.Errorf("value already exists for id %x", target)
}

// AddOrTouch adds a value if it is new, when the key exists, it touches it. In all cases it updates seq/cas/k/sig.
func (t *ValueStore) AddOrTouch(target string, value string, seq, cas int, k, sig []byte) error {
	if t.Add(target, value) != nil {
		t.values[target].Touch()
	}
	t.values[target].K = k
	t.values[target].Sig = sig
	t.values[target].Seq = seq
	t.values[target].Cas = cas
	return nil
}

// Get return the value stored for given target.
func (t *ValueStore) Get(target string) (*StoredValue, bool) {
	if x, ok := t.values[target]; ok {
		return x, ok
	}
	return nil, false
}

// Contains return true when the target exists.
func (t *ValueStore) Contains(target string) bool {
	if _, ok := t.values[target]; ok {
		return ok
	}
	return false
}

// Rm a value for given target.
func (t *ValueStore) Rm(target string) bool {
	if _, ok := t.values[target]; ok {
		delete(t.values, target)
		return ok
	}
	return false
}

//Clear the storage.
func (t *ValueStore) Clear() {
	t.values = map[string]*StoredValue{}
}

// TSValueStore is TS of ValueStore.
type TSValueStore struct {
	store *ValueStore
	mu    *sync.RWMutex
}

// NewTSValueStore is a constructor of TS store.
func NewTSValueStore() *TSValueStore {
	return &TSValueStore{
		store: NewValueStore(),
		mu:    &sync.RWMutex{},
	}
}

// Add a value for given target, it returns an error if the value already exist.
func (t *TSValueStore) Add(target string, value string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.store.Add(target, value)
}

// AddOrTouch adds a value if it is new, when the key exists, it touches it. In all cases it updates seq/cas/k/sig.
func (t *TSValueStore) AddOrTouch(target string, value string, seq, cas int, k, sig []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.store.AddOrTouch(target, value, seq, cas, k, sig)
}

// Get return the value stored for given target.
func (t *TSValueStore) Get(target string) (*StoredValue, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.store.Get(target)
}

// Contains return true when the target exists.
func (t *TSValueStore) Contains(target string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.store.Contains(target)
}

// Rm a value for given target.
func (t *TSValueStore) Rm(target string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.store.Rm(target)
}

// Transact operations.
func (t *TSValueStore) Transact(f func(*ValueStore)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	f(t.store)
}

//Clear the storage.
func (t *TSValueStore) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.store.Clear()
}
