package dht

import (
	"encoding/hex"

	"github.com/mh-cbon/dht/bucket"
)

// ClosestLocation finds closest peers for given target in the bootstrap table.
func (d *DHT) ClosestLocation(hexTarget string, n uint) (closests []bucket.ContactIdentifier, err error) {
	target, e := hex.DecodeString(hexTarget)
	if e != nil {
		return nil, e
	}
	return d.rpc.ClosestLocation(target, n)
}

// ClosestPeers finds closest peers for given target in a peers lookup table.
func (d *DHT) ClosestPeers(hexTarget string, n uint) (closests []bucket.ContactIdentifier, err error) {
	target, e := hex.DecodeString(hexTarget)
	if e != nil {
		return nil, e
	}
	return d.rpc.ClosestPeers(target, n)
}

// ClosestStores find closest stores for given target in a store lookup table.
func (d *DHT) ClosestStores(hexTarget string, n uint) (closest []bucket.ContactIdentifier, err error) {
	target, e := hex.DecodeString(hexTarget)
	if e != nil {
		return nil, e
	}
	return d.rpc.ClosestStores(target, n)
}
