package rpc

import (
	"fmt"

	"github.com/mh-cbon/dht/bucket"
)

//MakeTable is an helper to create a configured table and registres its ping handler.
func (k *KRPC) MakeTable(target []byte) *bucket.TSBucket {
	bucketTable := bucket.NewTS(target)
	bucketTable.Configure(k.k, k.concurrency)
	bucketTable.OnPing(k.handleTablePing(bucketTable))
	return bucketTable
}

// ClosestLocation returns the closest peers available in the bootstrap table. You must bootstrap before calling this func.
func (k *KRPC) ClosestLocation(target []byte, n uint) ([]bucket.ContactIdentifier, error) {
	if k.bootstrap == nil {
		return nil, fmt.Errorf("table not found for %x, run a lookup first", string(target))
	}
	return k.bootstrap.Closest(target, n), nil
}

// getTableForPeers creates or returns the table for given target.
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

// ClosestPeers returns the closest peers available for given target. You must perform a lookup of given target before.
// ClosestPeers for given target.
func (k *KRPC) ClosestPeers(target []byte, n uint) ([]bucket.ContactIdentifier, error) {
	table, found := k.lookupTableForPeers.Get(target)
	if found == false {
		return nil, fmt.Errorf("table not found for %x, run a lookup first", string(target))
	}
	return table.Closest(target, n), nil
}

// getTableForStore creates or returns the table for given target.
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

// ClosestStores returns the closest peers available for given target. You must perform a lookup of given target before.
func (k *KRPC) ClosestStores(target []byte, n uint) ([]bucket.ContactIdentifier, error) {
	table, found := k.lookupTableForStores.Get(target)
	if found == false {
		return nil, fmt.Errorf("table not found for %x, run a lookup first", string(target))
	}
	return table.Closest(target, n), nil
}

// ReleaseTableByID releases resources associated with given lookup id.
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
