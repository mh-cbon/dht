package rpc

import (
	"encoding/hex"
	"fmt"
	"net"

	"github.com/mh-cbon/dht/bucket"
	"github.com/mh-cbon/dht/crypto"
	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/socket"
	boom "github.com/tylertreat/BoomFilters"
)

// SeedFunc is the signature func to seed a table
type SeedFunc func(bucket.ContactIdentifier, func(kmsg.Msg)) (*socket.Tx, error)

//MakeTable is an helper to create a configured table.
func (k *KRPC) seedTable(target []byte, boostrapTable *bucket.TSBucket, table *bucket.TSBucket, seedF SeedFunc) error {
	startupContacts := boostrapTable.Closest(target, 8)
	if len(startupContacts) == 0 {
		return fmt.Errorf("Bootstrap table failed %x %v", string(target), startupContacts)
	}
	errs := k.BatchNodes(startupContacts, func(contact bucket.ContactIdentifier, done chan<- error) error {
		_, qErr := seedF(contact, func(res kmsg.Msg) {
			if res.E == nil && res.Y == "r" && res.R != nil {
				for _, node := range k.goodNodes(res.R.Nodes) {
					table.Add(node)
				}
			}
			done <- res.E
		})
		return qErr
	})
	if len(errs) == len(startupContacts) {
		return fmt.Errorf("All bootstrap nodes failed: %v", errs)
	}
	return nil
}

// LookupPeers for given target.
func (k *KRPC) LookupPeers(target []byte, boostrapTable *bucket.TSBucket) error {

	table, tableErr := k.getTableForPeers(target)
	if tableErr != nil {
		return tableErr
	}

	if boostrapTable == nil {
		boostrapTable = k.bootstrap
	}

	if boostrapTable == nil {
		return fmt.Errorf("Impossible to run a lookup: the bootstrap table is not ready")
	}

	seedErr := k.seedTable(target, boostrapTable, table, func(contact bucket.ContactIdentifier, done func(kmsg.Msg)) (*socket.Tx, error) {
		return k.FindNode(contact.GetAddr(), target, done)
	})
	if seedErr != nil {
		return seedErr
	}

	hops := 0
	queriedNodes := boom.NewBloomFilter(1500, 0.5)
	for {
		ncontacts := table.Closest(target, 8)
		contacts := k.notQueried(queriedNodes, ncontacts)
		if len(contacts) == 0 {
			break
		}
		hops++
		k.BatchNodes(contacts, func(contact bucket.ContactIdentifier, done chan<- error) error {
			_, qErr := k.FindNode(contact.GetAddr(), target, func(res kmsg.Msg) {
				if res.E == nil && res.Y == "r" && res.R != nil && res.SenderID() == string(contact.GetID()) {
					for _, node := range k.goodNodes(res.R.Nodes) {
						table.Add(node)
					}
				}
				done <- res.E
			})
			return qErr
		})
	}

	return nil
}

// LookupStores for given target.
func (k *KRPC) LookupStores(target []byte, boostrapTable *bucket.TSBucket) error {

	table, tableErr := k.getTableForStore(target)
	if tableErr != nil {
		return tableErr
	}

	if boostrapTable == nil {
		boostrapTable = k.bootstrap
	}

	if boostrapTable == nil {
		return fmt.Errorf("Impossible to run a lookup: the bootstrap table is not ready")
	}

	seedErr := k.seedTable(target, boostrapTable, table, func(contact bucket.ContactIdentifier, done func(kmsg.Msg)) (*socket.Tx, error) {
		return k.Get(contact.GetAddr(), target, done)
	})
	if seedErr != nil {
		return seedErr
	}

	hops := 0
	queriedNodes := boom.NewBloomFilter(3500, 0.5)
	for {
		ncontacts := table.Closest(target, 8)
		contacts := k.notQueried(queriedNodes, ncontacts)
		if len(contacts) == 0 {
			break
		}
		hops++
		k.BatchNodes(contacts, func(contact bucket.ContactIdentifier, done chan<- error) error {
			_, qErr := k.Get(contact.GetAddr(), target, func(res kmsg.Msg) {
				if res.E == nil && res.Y == "r" && res.R != nil && res.SenderID() == string(contact.GetID()) {
					for _, node := range k.goodNodes(res.R.Nodes) {
						table.Add(node)
					}
				}
				done <- res.E
			})
			return qErr
		})
	}

	return nil
}

//testPut checks given addr implemens get/put.
func (k *KRPC) testGetPut(addr *net.UDPAddr, onResponse func(kmsg.Msg)) error {
	v1 := "check"
	h1 := crypto.HashSha1(v1)
	t1, err := hex.DecodeString(h1)
	if err != nil {
		return err
	}

	_, err = k.Get(addr, t1, func(res kmsg.Msg) {
		if res.E != nil || res.R == nil || res.R.Token == "" {
			onResponse(res)
		} else {
			_, err2 := k.Put(addr, v1, res.R.Token, func(res1 kmsg.Msg) {
				if res1.E != nil {
					onResponse(res1)
				} else {
					_, err3 := k.Get(addr, t1, onResponse)
					if err3 != nil {
						onResponse(kmsg.Msg{E: &kmsg.Error{Code: 999, Msg: err3.Error()}})
					}
				}
			})
			if err2 != nil {
				onResponse(kmsg.Msg{E: &kmsg.Error{Code: 999, Msg: err2.Error()}})
			}
		}
	})
	return err
}
