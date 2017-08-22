package dht

import (
	"encoding/hex"

	"github.com/mh-cbon/dht/bucket"
)

// ClosestLocation uses the bootstrap table to find the closests nodes of given target.
func (d *DHT) ClosestLocation(hexTarget string, n uint) (closests []bucket.ContactIdentifier, err error) {
	target, e := hex.DecodeString(hexTarget)
	if e != nil {
		return nil, e
	}
	return d.rpc.ClosestLocation(target, n)
}

// ClosestPeers creates or re uses a lookup table finds closest peers using get_peers query. It fails if the hexTarget was not lookup before.
func (d *DHT) ClosestPeers(hexTarget string, n uint) (closests []bucket.ContactIdentifier, err error) {
	target, e := hex.DecodeString(hexTarget)
	if e != nil {
		return nil, e
	}
	// todo: automatically lookup if never done.
	return d.rpc.ClosestPeers(target, n)
}

// ClosestStores creates or re uses a lookup table finds closest peers using get query. It fails if the hexTarget was not lookup before.
func (d *DHT) ClosestStores(hexTarget string, n uint) (closest []bucket.ContactIdentifier, err error) {
	target, e := hex.DecodeString(hexTarget)
	if e != nil {
		return nil, e
	}
	// todo: automatically lookup if never done.
	return d.rpc.ClosestStores(target, n)
}
