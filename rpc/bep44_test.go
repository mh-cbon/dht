package rpc

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/socket"
)

func TestBep44(t *testing.T) {
	timeout := time.Millisecond * 10
	port := 9676
	// wantErr := func(t *testing.T, wanted error, got error) {
	// 	if got == nil {
	// 		t.Errorf("Wanted err=%v, got err=%v", wanted, got)
	// 		t.FailNow()
	// 		return
	// 	}
	// 	if wanted.Error() != got.Error() {
	// 		t.Errorf("Wanted err=%v, got err=%v", wanted.Error(), got.Error())
	// 		t.FailNow()
	// 		return
	// 	}
	// }
	// rejectErr := func(t *testing.T, got error) {
	// 	if got != nil {
	// 		t.Errorf("Wanted err=%v, got err=%v", nil, got)
	// 		t.FailNow()
	// 		return
	// 	}
	// }
	makeSocket := func(name string, ip string, timeout time.Duration) *socket.RPC {
		addr := fmt.Sprintf("%v:%v", ip, port)
		port++
		return socket.New(socket.RPCConfig{}.WithID(makID(name)).WithTimeout(timeout).WithAddr(addr))
	}
	t.Run("should make proper immutable get request", func(t *testing.T) {
		alice := makeSocket("alice", "127.0.0.1", timeout)
		go alice.MustListen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			if msg.A == nil {
				t.Errorf("wanted msg.A!=nil, got=%v", nil)
			} else if msg.A.Target != "abcd" {
				t.Errorf("wanted msg.A.Target=%v, got=%v", "abcd", msg.A.Target)
			} else if msg.A.ID != "bob\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" {
				t.Errorf("wanted msg.A.Id=%v, got=%#v", "bob", msg.A.ID)
			} else if len(msg.A.K) != 0 {
				t.Errorf("wanted msg.A.K=%v, got=%v", []byte{}, msg.A.K)
			} else if len(msg.A.Salt) != 0 {
				t.Errorf("wanted msg.A.Salt=%v, got=%v", "", msg.A.Salt)
			} else if len(msg.A.Sign) != 0 {
				t.Errorf("wanted msg.A.Sign=%v, got=%v", []byte{}, msg.A.Sign)
			} else if msg.A.Seq != 0 {
				t.Errorf("wanted msg.A.Seq=%v, got=%v", 0, msg.A.Seq)
			} else if msg.A.Cas != 0 {
				t.Errorf("wanted msg.A.Cas=%v, got=%v", 0, msg.A.Cas)
			} else if msg.A.Token != "" {
				t.Errorf("wanted msg.A.Token=%v, got=%v", "", msg.A.Token)
			}
			return alice.Respond(remote, msg.T, kmsg.Return{V: "hello"})
		})
		defer alice.Close()

		bob := makeSocket("bob", "127.0.0.1", timeout)
		go bob.Listen(nil)
		defer bob.Close()
		bobRPC := New(bob, KRPCConfig{})

		addrs := []*net.UDPAddr{
			alice.Addr(),
		}
		id := []byte("abcd")
		bobRPC.BatchAddrs(addrs, func(remote *net.UDPAddr, done chan<- error) error {
			_, qerr := bobRPC.Get(remote, id[:], func(res kmsg.Msg) {
				done <- res.E
			})
			return qerr
		})
	})
	t.Run("should make proper immutable put request", func(t *testing.T) {
		alice := makeSocket("alice", "127.0.0.1", timeout)
		go alice.MustListen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			if msg.A == nil {
				t.Errorf("wanted msg.A!=nil, got=%v", nil)
			} else if msg.A.Target != "" {
				t.Errorf("wanted msg.A.Target=%v, got=%v", "", msg.A.Target)
			} else if msg.A.ID != "bob\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" {
				t.Errorf("wanted msg.A.Id=%v, got=%#v", "bob", msg.A.ID)
			} else if len(msg.A.K) != 0 {
				t.Errorf("wanted msg.A.K=%v, got=%v", []byte{}, msg.A.K)
			} else if len(msg.A.Salt) != 0 {
				t.Errorf("wanted msg.A.Salt=%v, got=%v", "", msg.A.Salt)
			} else if len(msg.A.Sign) != 0 {
				t.Errorf("wanted msg.A.Sign=%v, got=%v", []byte{}, msg.A.Sign)
			} else if msg.A.Seq != 0 {
				t.Errorf("wanted msg.A.Seq=%v, got=%v", 0, msg.A.Seq)
			} else if msg.A.Cas != 0 {
				t.Errorf("wanted msg.A.Cas=%v, got=%v", 0, msg.A.Cas)
			} else if msg.A.Token != "writeToken" {
				t.Errorf("wanted msg.A.Token=%v, got=%v", "writeToken", msg.A.Token)
			} else if msg.A.V != "hello" {
				t.Errorf("wanted msg.A.V=%v, got=%v", "hello", msg.A.V)
			}
			return alice.Respond(remote, msg.T, kmsg.Return{V: "hello"})
		})
		defer alice.Close()

		bob := makeSocket("bob", "127.0.0.1", timeout)
		go bob.Listen(nil)
		defer bob.Close()
		bobRPC := New(bob, KRPCConfig{})

		addrs := []*net.UDPAddr{
			alice.Addr(),
		}
		bobRPC.BatchAddrs(addrs, func(remote *net.UDPAddr, done chan<- error) error {
			_, qerr := bobRPC.Put(remote, "hello", "writeToken", func(res kmsg.Msg) {
				done <- res.E
			})
			return qerr
		})
	})
	t.Run("should make proper mutable get request", func(t *testing.T) {
		alice := makeSocket("alice", "127.0.0.1", timeout)
		go alice.MustListen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			if msg.A == nil {
				t.Errorf("wanted msg.A!=nil, got=%v", nil)
			} else if msg.A.Target != "abcd" {
				t.Errorf("wanted msg.A.Target=%v, got=%v", "abcd", msg.A.Target)
			} else if msg.A.ID != "bob\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" {
				t.Errorf("wanted msg.A.Id=%v, got=%#v", "bob", msg.A.ID)
			} else if len(msg.A.K) != 0 {
				t.Errorf("wanted msg.A.K=%v, got=%v", []byte{}, msg.A.K)
			} else if msg.A.Salt != "" {
				t.Errorf("wanted msg.A.Salt=%v, got=%v", "", msg.A.Salt)
			} else if len(msg.A.Sign) != 0 {
				t.Errorf("wanted msg.A.Sign=%v, got=%v", []byte{}, msg.A.Sign)
			} else if msg.A.Seq != 2 {
				t.Errorf("wanted msg.A.Seq=%v, got=%v", 2, msg.A.Seq)
			} else if msg.A.Cas != 0 {
				t.Errorf("wanted msg.A.Cas=%v, got=%v", 0, msg.A.Cas)
			} else if msg.A.Token != "" {
				t.Errorf("wanted msg.A.Token=%v, got=%v", "", msg.A.Token)
			}
			return alice.Respond(remote, msg.T, kmsg.Return{V: "hello"})
		})
		defer alice.Close()

		bob := makeSocket("bob", "127.0.0.1", timeout)
		go bob.Listen(nil)
		defer bob.Close()
		bobRPC := New(bob, KRPCConfig{})

		addrs := []*net.UDPAddr{
			alice.Addr(),
		}
		id := []byte("abcd")
		bobRPC.BatchAddrs(addrs, func(remote *net.UDPAddr, done chan<- error) error {
			_, qerr := bobRPC.MGet(remote, id[:], 2, func(res kmsg.Msg) {
				done <- res.E
			})
			return qerr
		})
	})
	t.Run("should make proper mutable put request", func(t *testing.T) {
		alice := makeSocket("alice", "127.0.0.1", timeout)
		go alice.MustListen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			if msg.A == nil {
				t.Errorf("wanted msg.A!=nil, got=%v", nil)
			} else if msg.A.Target != "" {
				t.Errorf("wanted msg.A.Target=%v, got=%v", "", msg.A.Target)
			} else if msg.A.ID != "bob\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" {
				t.Errorf("wanted msg.A.Id=%v, got=%#v", "bob", msg.A.ID)
			} else if bytes.Equal([]byte("pbk"), msg.A.K) == false {
				t.Errorf("wanted msg.A.K=%v, got=%v", []byte("pbk"), msg.A.K)
			} else if msg.A.Salt != "salt" {
				t.Errorf("wanted msg.A.Salt=%v, got=%v", "salt", msg.A.Salt)
			} else if bytes.Equal([]byte("sign"), msg.A.Sign) == false {
				t.Errorf("wanted msg.A.Sign=%v, got=%v", []byte("sign"), msg.A.Sign)
			} else if msg.A.Seq != 2 {
				t.Errorf("wanted msg.A.Seq=%v, got=%v", 2, msg.A.Seq)
			} else if msg.A.Cas != 3 {
				t.Errorf("wanted msg.A.Cas=%v, got=%v", 3, msg.A.Cas)
			} else if msg.A.Token != "writeToken" {
				t.Errorf("wanted msg.A.Token=%v, got=%v", "writeToken", msg.A.Token)
			} else if msg.A.V != "value" {
				t.Errorf("wanted msg.A.V=%v, got=%v", "value", msg.A.V)
			}
			return alice.Respond(remote, msg.T, kmsg.Return{V: "hello"})
		})
		defer alice.Close()

		bob := makeSocket("bob", "127.0.0.1", timeout)
		go bob.Listen(nil)
		defer bob.Close()
		bobRPC := New(bob, KRPCConfig{})

		addrs := []*net.UDPAddr{
			alice.Addr(),
		}
		bobRPC.BatchAddrs(addrs, func(remote *net.UDPAddr, done chan<- error) error {
			_, qerr := bobRPC.MPut(remote, "value", []byte("pbk"), []byte("sign"), 2, 3, "salt", "writeToken", func(res kmsg.Msg) {
				done <- res.E
			})
			return qerr
		})
	})
	// t.Run("should blank a response get token when the node id is insecure", func(t *testing.T) {
	// 	alice := makeSocket("alice", "127.0.0.1", timeout)
	// 	go alice.MustListen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
	// 		return alice.Respond(remote, msg.T, kmsg.Return{Token: "to blank"})
	// 	})
	// 	defer alice.Close()
	//
	// 	bob := makeSocket("bob", "127.0.0.1", timeout)
	// 	go bob.Listen(nil)
	// 	defer bob.Close()
	// 	bobRPC := New(bob, KRPCConfig{})
	//
	// 	id := []byte("abcd")
	// 	bobRPC.Get(alice.Addr(), id[:], func(res kmsg.Msg) {
	// 		if res.R.Token != "" {
	// 			t.Errorf("Wanted res.R.Token=%v, got=%v", "", res.R.Token)
	// 		}
	// 	})
	// 	<-time.After(timeout * 2)
	// })
}
