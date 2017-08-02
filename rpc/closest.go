package rpc

import (
	"fmt"

	"github.com/mh-cbon/dht/bucket"
)

//MakeTable is an helper to create a configured table.
func (k *KRPC) MakeTable(target []byte) *bucket.TSBucket {
	bucketTable := bucket.NewTS(target)
	bucketTable.Configure(k.config.k, k.config.concurrency)
	bucketTable.OnPing(k.handleTablePing(bucketTable))
	return bucketTable
}

// ClosestLocation for given target.
func (k *KRPC) ClosestLocation(target []byte, n uint) ([]bucket.ContactIdentifier, error) {
	if k.bootstrap == nil {
		return nil, fmt.Errorf("table not found for %x, run a lookup first", string(target))
	}
	return k.bootstrap.Closest(target, n), nil
}

// ClosestStores find closest stores for given target via get messages.
func (k *KRPC) getTableForPeers(target []byte) (table *bucket.TSBucket, err error) {
	var found bool
	x, found := k.lookupTableForPeers.Get(target)
	table = x
	if found == false {
		table = k.MakeTable(target)
		k.lookupTableForPeers.Add(target, table)
	}
	return table, nil
}

// ClosestPeers for given target.
func (k *KRPC) ClosestPeers(target []byte, n uint) ([]bucket.ContactIdentifier, error) {
	table, found := k.lookupTableForPeers.Get(target)
	if found == false {
		return nil, fmt.Errorf("table not found for %x, run a lookup first", string(target))
	}
	return table.Closest(target, n), nil
}

// ClosestStores find closest stores for given target via get messages.
func (k *KRPC) getTableForStore(target []byte) (table *bucket.TSBucket, err error) {
	var found bool
	x, found := k.lookupTableForStores.Get(target)
	table = x
	if found == false {
		table = k.MakeTable(target)
		k.lookupTableForStores.Add(target, table)
	}
	return table, nil
}

// ClosestStores for given target.
func (k *KRPC) ClosestStores(target []byte, n uint) ([]bucket.ContactIdentifier, error) {
	table, found := k.lookupTableForStores.Get(target)
	if found == false {
		return nil, fmt.Errorf("table not found for %x, run a lookup first", string(target))
	}
	return table.Closest(target, n), nil
}

// ReleaseTableByID releases resources associated with this lookup id.
func (k *KRPC) ReleaseTableByID(target []byte) error {
	if table, found := k.lookupTableForStores.Get(target); found {
		table.Clear()
		k.lookupTableForStores.Rm(target)
		return nil
	}
	if table, found := k.lookupTableForPeers.Get(target); found {
		table.Clear()
		k.lookupTableForPeers.Rm(target)
		return nil
	}
	return fmt.Errorf("table not found for %x", string(target))
}
