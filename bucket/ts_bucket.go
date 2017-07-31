package bucket

import (
	"net"
	"sync"
)

// TSBucket is thread safe bucket.
type TSBucket struct {
	bucket *KBucket
	mu     *sync.RWMutex
}

// NewTS returns a thread safe bucket.
func NewTS(targetID []byte) *TSBucket {
	ret := &TSBucket{
		New(targetID),
		&sync.RWMutex{},
	}
	return ret
}

// TSFrom returns a thread safe bucket from an existing table.
func TSFrom(table *KBucket) *TSBucket {
	ret := &TSBucket{
		table,
		&sync.RWMutex{},
	}
	return ret
}

// Configure the bucket. NTS
func (k *TSBucket) Configure(numberOfNodesPerKBucket, numberOfNodesToPing int) {
	k.bucket.Configure(numberOfNodesPerKBucket, numberOfNodesToPing)
}

//OnPing registers a ping function. NTS
func (k *TSBucket) OnPing(p PingFunc) {
	k.bucket.OnPing(p)
}

//OnAdded registers a added function. NTS
func (k *TSBucket) OnAdded(p AddFunc) {
	k.bucket.OnAdded(p)
}

//OnRemoved registers a removed function. NTS
func (k *TSBucket) OnRemoved(p RemoveFunc) {
	k.bucket.OnRemoved(p)
}

//OnUpdated registers an updated function. NTS
func (k *TSBucket) OnUpdated(p UpdateFunc) {
	k.bucket.OnUpdated(p)
}

// Clear the bucket. NTS
func (k *TSBucket) Clear() {
	k.bucket.Clear()
}

// ToArray export the table
func (k *TSBucket) ToArray() (ret []ContactIdentifier) {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.bucket.ToArray()
}

// AddMany contacts to the bucket. Stop on first error.
func (k *TSBucket) AddMany(contacts ...*KContact) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.bucket.AddMany(contacts...)
}

// Add contact: *required* the contact object to add
func (k *TSBucket) Add(contact ContactIdentifier) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.bucket.Add(contact)
}

// Closest contacts for given id.
func (k *TSBucket) Closest(id []byte, n uint) []ContactIdentifier {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.bucket.Closest(id, n)
}

// Count contacts.
func (k *TSBucket) Count() int {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.bucket.Count()
}

// Get a contact for given id.
func (k *TSBucket) Get(id []byte) ContactIdentifier {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.bucket.Get(id)
}

// Remove a contact with given id.
func (k *TSBucket) Remove(id []byte) ContactIdentifier {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.bucket.Remove(id)
}

// RemoveByAddr a contact with given addr.
func (k *TSBucket) RemoveByAddr(addr *net.UDPAddr) []ContactIdentifier {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.bucket.RemoveByAddr(addr)
}
