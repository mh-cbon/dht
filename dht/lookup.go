package dht

import (
	"encoding/hex"

	"github.com/mh-cbon/dht/bucket"
)

// LookupPeers perform a lookup with "get_peers" messages of given hexTarget using given bootstrap table.
func (k *DHT) LookupPeers(hexTarget string, boostrapTable *bucket.TSBucket) error {
	target, e := hex.DecodeString(hexTarget)
	if e != nil {
		return e
	}
	return k.rpc.LookupPeers(target, boostrapTable)
}

// LookupStores  perform a lookup with "get" messages of given hexTarget using given bootstrap table.
func (k *DHT) LookupStores(hexTarget string, boostrapTable *bucket.TSBucket) error {
	target, e := hex.DecodeString(hexTarget)
	if e != nil {
		return e
	}
	return k.rpc.LookupStores(target, boostrapTable)
}

// ReleaseLookupTable releases resources associated with given hexTarget.
func (k *DHT) ReleaseLookupTable(hexTarget string) error {
	target, e := hex.DecodeString(hexTarget)
	if e != nil {
		return e
	}
	return k.rpc.ReleaseTableByID(target)
}
