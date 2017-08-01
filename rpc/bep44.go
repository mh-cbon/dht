package rpc

import (
	"fmt"
	"net"

	"github.com/anacrolix/torrent/bencode"
	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/socket"
	"golang.org/x/crypto/ed25519"
)

// QGet is the "get" query verb.
var QGet = "get"

// Get sends an immutable "get" query.
func (k *KRPC) Get(addr *net.UDPAddr, target []byte, onResponse func(kmsg.Msg)) (*socket.Tx, error) {
	a := map[string]interface{}{"target": target}
	// bep42: guards against bad node id
	return k.Query(addr, QGet, a, SecuredResponseOnly(addr, func(msg kmsg.Msg) {
		if msg.E != nil {
			//tdo: check about that with stat store
			// avoid putting that node into the lookup tables.
			// k.addBadNode(addr)
			k.lookupTableForStores.RemoveNode(addr)
		}
		if onResponse != nil {
			onResponse(msg)
		}
	}))
}

//CheckGetResponse for given seq and salt.
func CheckGetResponse(msg kmsg.Msg, pbK []byte, seq int, salt string) *kmsg.Error {
	if msg.R.Seq < seq {
		return &kmsg.ErrorSeqLessThanCurrent
	}
	return CheckSign(pbK, msg.R.Sign, msg.R.V, msg.R.Seq, salt)
}

//CheckPutQuery for given seq and salt.
func CheckPutQuery(msg kmsg.Msg) *kmsg.Error {
	if len(msg.A.V) == 0 {
		return &kmsg.ErrorVTooShort
	}
	if len(msg.A.V) > 999 {
		return &kmsg.ErrorVTooLong
	}
	if len(msg.A.K) > 0 {
		if len(msg.A.Salt) > 64 {
			return &kmsg.ErrorSaltTooLong
		}
		return CheckSign(msg.A.K, msg.A.Sign, msg.A.V, msg.A.Seq, msg.A.Salt)
	}
	return nil
}

//CheckSign value/sig/pbk ect.
func CheckSign(pbK, sig []byte, v string, seq int, salt string) *kmsg.Error {
	if len(pbK) == 0 {
		return &kmsg.ErrorNoK
	}
	if len(sig) == 0 {
		return &kmsg.ErrorInvalidSig
	}
	hash, err := EncodePutValue(v, salt, seq)
	if err != nil {
		return &kmsg.ErrorInternalIssue
	}
	if ed25519.Verify(pbK, []byte(hash), sig) == false {
		return &kmsg.ErrorInvalidSig
	}
	return nil
}

// EncodePutValue encodes v+salt+seq+salt before hashing.
func EncodePutValue(val, salt string, seq int) (string, error) {
	var ret string
	v, err := bencode.Marshal(val)
	if err != nil {
		return "", err
	}
	ret = fmt.Sprintf(":v%v", string(v))

	se, err := bencode.Marshal(seq)
	if err != nil {
		return "", err
	}
	ret = fmt.Sprintf("3:seq%v1%v", string(se), ret)

	if salt != "" {
		sa, err := bencode.Marshal(salt)
		if err != nil {
			return "", err
		}
		ret = fmt.Sprintf("4:salt%v%v", string(sa), ret)
	}
	return ret, nil
}

// MGet sends an mutable "get" query.
func (k *KRPC) MGet(addr *net.UDPAddr, target []byte, seq int, onResponse func(kmsg.Msg)) (*socket.Tx, error) {
	a := map[string]interface{}{"target": target, "seq": seq}
	// bep42: guards against bad node id
	return k.Query(addr, QGet, a, SecuredResponseOnly(addr, onResponse))
}

// QPut is the "put" query verb.
var QPut = "put"

// Put sends an immutable "put" query.
func (k *KRPC) Put(addr *net.UDPAddr, value, writeToken string, onResponse func(kmsg.Msg)) (*socket.Tx, error) {
	a := map[string]interface{}{
		"token": writeToken,
		"v":     value,
	}
	return k.Query(addr, QPut, a, onResponse)
}

// MPut sends an mutable "put" query.
func (k *KRPC) MPut(addr *net.UDPAddr, value string, pbk []byte, sig []byte, seq int, cas int, salt string, writeToken string, onResponse func(kmsg.Msg)) (*socket.Tx, error) {
	a := map[string]interface{}{
		"token": writeToken,
		"v":     value,
		"sig":   sig,
		"k":     pbk,
		"seq":   seq,
		"cas":   cas,
		"salt":  salt,
	}
	return k.Query(addr, QPut, a, onResponse)
}
