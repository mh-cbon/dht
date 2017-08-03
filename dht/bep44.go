package dht

import (
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/mh-cbon/dht/bucket"
	"github.com/mh-cbon/dht/crypto"
	"github.com/mh-cbon/dht/ed25519"
	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/rpc"
	"github.com/mh-cbon/dht/socket"
)

// OnGet responds to a get query.
// If the queried node has a value for the infohash, it is returned in a key "v" as a string.
// If the queried node has no value for the infohash,
// a key "nodes" is returned containing the K nodes in the queried nodes routing table closest to the infohash supplied in the query.
// In either case a "token" key is also included in the return value.
// The token value is a required argument for a future put query.
// The token value should be a short binary string.
func (d *DHT) OnGet(msg kmsg.Msg, remote *net.UDPAddr) error {
	if msg.A == nil {
		return fmt.Errorf("bad message no A: %v", msg)
	}

	target := msg.A.Target
	if len(target) != 20 {
		return fmt.Errorf("bad target len: %v", msg)
	}
	hexTarget := HexFromBytes([]byte(target))
	// x := hex.EncodeToString([]byte(target))
	// hexTarget := string(x)

	v := ""
	var k []byte
	var sig []byte
	var seq int
	d.bep44ValueStore.Transact(func(store *ValueStore) {
		// If a stored item exists,
		log.Println("target", target)
		log.Println("hexTarget", hexTarget)
		if s, ok := store.Get(hexTarget); ok {
			// but its sequence number is less than or equal to the seq field
			// or its expired
			// then the k, v, and sig fields SHOULD be omitted from the response.
			isExpired := s.HasExpired(2 * time.Hour)
			if s.Seq >= msg.A.Seq && !isExpired {
				v = s.Value
				seq = s.Seq
				k = s.K
				sig = s.Sig
			} else if isExpired {
				store.Rm(hexTarget)
			}
		}
	})
	var nodes kmsg.CompactIPv4NodeInfo
	contacts, err := d.rpc.ClosestStores([]byte(target), 8)
	if err == nil && len(contacts) > 0 {
		nodes = []kmsg.NodeInfo{}
		for _, c := range contacts {
			nodes = append(nodes, rpc.NodeInfo(c))
		}
	}

	return d.Respond(remote, msg.T, kmsg.Return{
		Nodes: nodes,
		Token: d.tokenServer.CreateToken(remote),
		V:     v,
		K:     k,
		Sign:  sig,
	})
}

// Get issues a get request to addr for given target.
// It saves response write tokens for future put queries.
func (d *DHT) Get(addr *net.UDPAddr, hexTarget string, onResponse func(kmsg.Msg)) (*socket.Tx, error) {
	target, e := HexToBytes(hexTarget)
	if e != nil {
		return nil, e
	}
	return d.rpc.Get(addr, target, func(res kmsg.Msg) {
		if res.E == nil && res.R != nil && res.R.Token != "" {
			d.bep44TokenStore.SetToken(res.R.Token, addr)
		}
		if onResponse != nil {
			onResponse(res)
		}
	})
}

// GetAll issues a get request to the closest nodes for given target.
// It saves response write tokens for future put queries.
func (d *DHT) GetAll(hexTarget string, addrs ...*net.UDPAddr) (string, error) {
	target, e := HexToBytes(hexTarget)
	if e != nil {
		return "", e
	}

	mu := &sync.RWMutex{}
	ret := ""
	onResponse := func(addr *net.UDPAddr, res kmsg.Msg) error {
		if res.E == nil && res.R != nil {
			if res.R.V != "" {
				mu.Lock()
				ret = res.R.V
				mu.Unlock()
			}
		}
		return res.E
	}
	var gotErrs []error
	if len(addrs) > 0 {
		gotErrs = d.rpc.BatchAddrs(addrs, func(remote *net.UDPAddr, done chan<- error) (*socket.Tx, error) {
			return d.Get(remote, hexTarget, func(res kmsg.Msg) {
				done <- onResponse(remote, res)
			})
		})
	} else {
		closests, err := d.ClosestStores(hexTarget, 8)
		if err != nil {
			return "", err
		}
		gotErrs = d.rpc.BatchNodes(closests, func(remote bucket.ContactIdentifier, done chan<- error) (*socket.Tx, error) {
			return d.Get(remote.GetAddr(), hexTarget, func(res kmsg.Msg) {
				done <- onResponse(remote.GetAddr(), res)
			})
		})
	}
	if ret == "" {
		return ret, fmt.Errorf("value not found for id %x got %v errors: %v", target, len(gotErrs), gotErrs)
	}
	return ret, nil
}

// MGet issues a mutable get request to addr for given target.
// It saves response write tokens for future put queries.
func (d *DHT) MGet(addr *net.UDPAddr, hexTarget string, pbk []byte, seq int, salt string, onResponse func(kmsg.Msg)) (*socket.Tx, error) {
	target, e := HexToBytes(hexTarget)
	if e != nil {
		return nil, e
	}

	return d.rpc.MGet(addr, target, seq, func(res kmsg.Msg) {
		if res.E == nil && res.R != nil {
			if res.R.Token != "" {
				d.bep44TokenStore.SetToken(res.R.Token, addr)
			}
			if res.R.V != "" {
				if err := rpc.CheckGetResponse(res, pbk, seq, salt); err != nil {
					res.R = nil
					res.E = err
				}
			}
		}
		if onResponse != nil {
			onResponse(res)
		}
	})
}

// MGetAll issues a mutable get request to the closest nodes for given target.
// It saves response write tokens for future put queries.
func (d *DHT) MGetAll(hexTarget string, pbk []byte, seq int, salt string, addrs ...*net.UDPAddr) (string, error) {
	target, e := HexToBytes(hexTarget)
	if e != nil {
		return "", e
	}

	mu := &sync.RWMutex{}
	ret := ""
	onResponse := func(addr *net.UDPAddr, res kmsg.Msg) error {
		if res.E == nil && res.R != nil {
			if res.R.V != "" {
				mu.Lock()
				ret = res.R.V
				defer mu.Unlock()
			}
		}
		return res.E
	}
	var gotErrs []error
	if len(addrs) > 0 {
		gotErrs = d.rpc.BatchAddrs(addrs, func(remote *net.UDPAddr, done chan<- error) (*socket.Tx, error) {
			return d.MGet(remote, hexTarget, pbk, seq, salt, func(res kmsg.Msg) {
				done <- onResponse(remote, res)
			})
		})
	} else {
		closests, err := d.ClosestStores(hexTarget, 8)
		if err != nil {
			return "", err
		}
		gotErrs = d.rpc.BatchNodes(closests, func(remote bucket.ContactIdentifier, done chan<- error) (*socket.Tx, error) {
			return d.MGet(remote.GetAddr(), hexTarget, pbk, seq, salt, func(res kmsg.Msg) {
				done <- onResponse(remote.GetAddr(), res)
			})
		})
	}
	if ret == "" {
		return ret, fmt.Errorf("value not found for id %x, got %v errors: %v", target, len(gotErrs), gotErrs)
	}
	return ret, nil
}

// OnPut responds to a put query.
// The queried node must verify that the token was previously sent to the same IP address as the querying node.
// Then the queried node should store the value in the "v" key of the message,
// under the infohash in its store of values.
func (d *DHT) OnPut(msg kmsg.Msg, remote *net.UDPAddr) error {

	token := msg.A.Token
	if token == "" {
		return fmt.Errorf("bad token len: %q", token)
	}
	if d.tokenServer.ValidToken(token, remote) == false {
		return d.Error(remote, msg.T, kmsg.ErrorBadToken)
	}

	if err := rpc.CheckPutQuery(msg); err != nil {
		return d.Error(remote, msg.T, *err)
	}

	var target string
	var retErr error
	d.bep44ValueStore.Transact(func(store *ValueStore) {
		if len(msg.A.K) > 0 {
			if s, ok := store.Get(target); ok {
				if msg.A.Seq < s.Seq {
					retErr = d.Error(remote, msg.T, kmsg.ErrorSeqLessThanCurrent)
					return
				}
				if msg.A.Cas != s.Cas {
					retErr = d.Error(remote, msg.T, kmsg.ErrorCasMismatch)
					return
				}
			}
			// for mutable message storeID is sha1(publicKey + salt)
			v := fmt.Sprintf("%v%v", string(msg.A.K), msg.A.Salt)
			target = ValueToHex(v)
		} else {
			// for immutable message storeID is the hash of the value
			target = ValueToHex(msg.A.V)
		}
		if err := store.AddOrTouch(target, msg.A.V); err != nil {
			d.Error(remote, msg.T, kmsg.ErrorInternalIssue)
			retErr = err // send the store error to the main loop for printing (we don t care if remote does not receive the 501 error)
		} else {
			retErr = d.Respond(remote, msg.T, kmsg.Return{})
		}
	})

	return retErr
}

// Put issues a put request to given addr for given target.
func (d *DHT) Put(addr *net.UDPAddr, value string, onResponse func(kmsg.Msg)) (*socket.Tx, error) {
	// Immutable items are stored under their SHA-1 hash,
	// and since they cannot be modified,
	// there is no need to authenticate the origin of them. This makes immutable items simple.
	writeToken := d.bep44TokenStore.GetToken(*addr)
	if writeToken == "" {
		_, target, e := ValueToHexAndByteString(value)
		if e != nil {
			return nil, e
		}
		return d.rpc.Get(addr, target, func(res kmsg.Msg) {
			if res.R != nil && res.E == nil && res.R.Token != "" {
				d.rpc.Put(addr, value, res.R.Token, onResponse)
			} else {
				onResponse(res)
			}
		})
	}
	return d.rpc.Put(addr, value, writeToken, onResponse)
}

// PutAll issues a put request to the closest nodes for given target.
func (d *DHT) PutAll(value string, addrs ...*net.UDPAddr) (string, error) {
	// Immutable items are stored under their SHA-1 hash,
	// and since they cannot be modified,
	// there is no need to authenticate the origin of them. This makes immutable items simple.
	hexTarget := ValueToHex(value)

	onResponse := func(addr *net.UDPAddr, done chan<- error) (*socket.Tx, error) {
		return d.Put(addr, value, func(res kmsg.Msg) {
			done <- res.E
		})
	}

	var errs []error
	addrsLen := len(addrs)
	if addrsLen > 0 {
		errs = d.rpc.BatchAddrs(addrs, onResponse)
	} else {
		closest, err := d.ClosestStores(hexTarget, 8)
		if err != nil {
			return hexTarget, err
		}
		addrsLen = len(closest)
		errs = d.rpc.BatchNodes(closest, func(remote bucket.ContactIdentifier, done chan<- error) (*socket.Tx, error) {
			return onResponse(remote.GetAddr(), done)
		})
	}
	if len(errs) == addrsLen {
		return hexTarget, fmt.Errorf("Put failed: %v %v", len(errs), errs)
	}
	return hexTarget, nil
}

// ValueToHexAndByteString turns a value to its hex and byte string.
func ValueToHexAndByteString(value string) (string, []byte, error) {
	h := ValueToHex(value)
	x, e := hex.DecodeString(h)
	if e != nil {
		return "", nil, e
	}
	return h, x, nil
}

// ValueToHex turns a value to its hex.
func ValueToHex(value string) string {
	return crypto.HashSha1(fmt.Sprintf("%v:%v", len(value), value))
}

// HexToBytes turns an hex to its byte string.
func HexToBytes(h string) ([]byte, error) {
	return hex.DecodeString(h)
}

// HexFromBytes turns a byte string into its hex.
func HexFromBytes(b []byte) string {
	return hex.EncodeToString(b)
}

// MutablePut holds mutable put message data.
type MutablePut struct {
	Val    string
	Salt   string
	Pbk    []byte
	Sign   []byte
	Seq    int
	Cas    int
	Target string
}

// PutFromPbk creates a mutable put message using a pbk.
func PutFromPbk(val string, salt string, pbk []byte, sign []byte, seq, cas int) (*MutablePut, error) {
	return &MutablePut{
		Val:    val,
		Salt:   salt,
		Pbk:    pbk,
		Sign:   sign,
		Seq:    seq,
		Cas:    cas,
		Target: crypto.HashSha1(string(pbk), salt),
	}, nil
}

// PutFromPvk creates a mutable put message using a pvk.
func PutFromPvk(val string, salt string, pvk ed25519.PrivateKey, seq, cas int) (*MutablePut, error) {
	encodedValue, err := rpc.EncodePutValue(val, salt, seq)
	if err != nil {
		return nil, err
	}
	pbk := ed25519.PublicKeyFromPvk(pvk)
	sign := ed25519.Sign(pvk, pbk, []byte(encodedValue))
	return PutFromPbk(val, salt, pbk, sign, seq, cas)
}

// MPut issues a mutable put request to addr for given target.
func (d *DHT) MPut(addr *net.UDPAddr, value *MutablePut, onResponse func(kmsg.Msg)) (*socket.Tx, error) {
	doPut := func(writeToken string) (*socket.Tx, error) {
		return d.rpc.MPut(addr, value.Val, value.Pbk, value.Sign, value.Seq, value.Cas, value.Salt, writeToken, onResponse)
	}
	writeToken := d.bep44TokenStore.GetToken(*addr)
	if writeToken == "" {
		return d.MGet(addr, value.Target, value.Pbk, value.Seq, value.Salt, func(res kmsg.Msg) {
			if res.E != nil || res.R == nil {
				onResponse(res)
			} else {
				doPut(res.R.Token)
			}
		})
	}
	return doPut(writeToken)
}

// MPutAll issues a mutable put request to the closest nodes for given target.
func (d *DHT) MPutAll(value *MutablePut, addrs ...*net.UDPAddr) error {

	hexTarget := value.Target

	onResponse := func(addr *net.UDPAddr, done chan<- error) (*socket.Tx, error) {
		return d.MPut(addr, value, func(res kmsg.Msg) {
			done <- res.E
		})
	}

	var errs []error
	addrsLen := len(addrs)
	if addrsLen > 0 {
		errs = d.rpc.BatchAddrs(addrs, onResponse)
	} else {
		// get closest nodes.
		closest, err := d.ClosestStores(hexTarget, 8)
		if err != nil {
			return err
		}
		errs = d.rpc.BatchNodes(closest, func(remote bucket.ContactIdentifier, done chan<- error) (*socket.Tx, error) {
			return onResponse(remote.GetAddr(), done)
		})
	}

	if len(errs) == addrsLen {
		return fmt.Errorf("Put failed: %v %v", len(errs), errs)
	}
	return nil
}
