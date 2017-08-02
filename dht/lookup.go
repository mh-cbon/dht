package dht

import (
	"encoding/hex"

	"github.com/mh-cbon/dht/bucket"
)

// LookupPeers for given target.
func (k *DHT) LookupPeers(hexTarget string, boostrapTable *bucket.TSBucket) error {
	target, e := hex.DecodeString(hexTarget)
	if e != nil {
		return e
	}
	return k.rpc.LookupPeers(target, boostrapTable)
}

// LookupStores for given target.
func (k *DHT) LookupStores(hexTarget string, boostrapTable *bucket.TSBucket) error {
	target, e := hex.DecodeString(hexTarget)
	if e != nil {
		return e
	}
	return k.rpc.LookupStores(target, boostrapTable)
}

// ReleaseLookupTable releases resources associated with this lookup id.
func (k *DHT) ReleaseLookupTable(hexTarget string) error {
	target, e := hex.DecodeString(hexTarget)
	if e != nil {
		return e
	}
	return k.rpc.ReleaseTableByID(target)
}
