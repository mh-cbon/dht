package rpc

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/socket"
)

func TestBep05(t *testing.T) {
	timeout := time.Millisecond * 10
	port := 9206
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
	newAddr := func() string {
		ip := "127.0.0.1"
		addr := fmt.Sprintf("%v:%v", ip, port)
		port++
		return addr
	}
	makeSocket := func(name string, timeout time.Duration) *socket.RPC {
		return socket.New(
			socket.RPCOpts.ID(string(makID(name))),
			socket.RPCOpts.WithTimeout(timeout),
			socket.RPCOpts.WithAddr(newAddr()),
		)
	}
	makeRPC := func(name string, timeout time.Duration) *KRPC {
		return New(
			KRPCOpts.ID(string(makID(name))),
			KRPCOpts.WithTimeout(timeout),
			KRPCOpts.WithAddr(newAddr()),
		)
	}
	t.Run("should make proper find_node request", func(t *testing.T) {
		alice := makeSocket("alice", timeout)
		go alice.MustListen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			if msg.A == nil {
				t.Errorf("wanted msg.A!=nil, got=%v", nil)
			} else if msg.A.Target != "abcd" {
				t.Errorf("wanted msg.A.Target=%v, got=%v", "abcd", msg.A.Target)
			}
			return alice.Respond(remote, msg.T, kmsg.Return{V: "hello"})
		})
		defer alice.Close()

		bob := makeRPC("bob", timeout)
		go bob.Listen(nil)
		defer bob.Close()

		addrs := []*net.UDPAddr{
			alice.GetAddr(),
		}
		id := []byte("abcd")
		bob.BatchAddrs(addrs, func(remote *net.UDPAddr, done chan<- error) (*socket.Tx, error) {
			return bob.FindNode(remote, id[:], func(res kmsg.Msg) {
				done <- res.E
			})
		})
	})
	t.Run("should make proper ping request", func(t *testing.T) {
		alice := makeSocket("alice", timeout)
		go alice.MustListen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			if msg.A == nil {
				t.Errorf("wanted msg.A!=nil, got=%v", nil)
			} else if msg.A.ID != "bob\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" {
				t.Errorf("wanted msg.A.Id=%v, got=%#v", "bob", msg.A.ID)
			}
			return alice.Respond(remote, msg.T, kmsg.Return{V: "hello"})
		})
		defer alice.Close()

		bob := makeRPC("bob", timeout)
		go bob.Listen(nil)
		defer bob.Close()

		addrs := []*net.UDPAddr{
			alice.GetAddr(),
		}
		bob.BatchAddrs(addrs, func(remote *net.UDPAddr, done chan<- error) (*socket.Tx, error) {
			return bob.Ping(remote, func(res kmsg.Msg) {
				done <- res.E
			})
		})
	})
	t.Run("should make proper get_peers request", func(t *testing.T) {
		alice := makeSocket("alice", timeout)
		go alice.MustListen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			if msg.A == nil {
				t.Errorf("wanted msg.A!=nil, got=%v", nil)
			} else if msg.A.ID != "bob\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" {
				t.Errorf("wanted msg.A.Id=%v, got=%#v", "bob", msg.A.ID)
			} else if msg.A.InfoHash != "abcd" {
				t.Errorf("wanted msg.A.InfoHash=%v, got=%v", "abcd", msg.A.InfoHash)
			}
			return alice.Respond(remote, msg.T, kmsg.Return{V: "hello"})
		})
		defer alice.Close()

		bob := makeRPC("bob", timeout)
		go bob.Listen(nil)
		defer bob.Close()

		addrs := []*net.UDPAddr{
			alice.GetAddr(),
		}
		id := []byte("abcd")
		bob.BatchAddrs(addrs, func(remote *net.UDPAddr, done chan<- error) (*socket.Tx, error) {
			return bob.GetPeers(remote, id[:], func(res kmsg.Msg) {
				done <- res.E
			})
		})
	})
	t.Run("should make proper announce_peer request", func(t *testing.T) {
		alice := makeSocket("alice", timeout)
		go alice.MustListen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			if msg.A == nil {
				t.Errorf("wanted msg.A!=nil, got=%v", nil)
			} else if msg.A.ID != "bob\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" {
				t.Errorf("wanted msg.A.Id=%v, got=%#v", "bob", msg.A.ID)
			} else if msg.A.InfoHash != "abcd" {
				t.Errorf("wanted msg.A.InfoHash=%v, got=%v", "abcd", msg.A.InfoHash)
			} else if msg.A.Token != "writeToken" {
				t.Errorf("wanted msg.A.Token=%v, got=%v", "writeToken", msg.A.Token)
			} else if msg.A.Port != 9090 {
				t.Errorf("wanted msg.A.Port=%v, got=%v", 9090, msg.A.Port)
			} else if msg.A.ImpliedPort != 1 {
				t.Errorf("wanted msg.A.ImpliedPort=%v, got=%v", 1, msg.A.ImpliedPort)
			}
			return alice.Respond(remote, msg.T, kmsg.Return{V: "hello"})
		})
		defer alice.Close()

		bob := makeRPC("bob", timeout)
		go bob.Listen(nil)
		defer bob.Close()

		addrs := []*net.UDPAddr{
			alice.GetAddr(),
		}
		id := []byte("abcd")
		bob.BatchAddrs(addrs, func(remote *net.UDPAddr, done chan<- error) (*socket.Tx, error) {
			return bob.AnnouncePeer(remote, id[:], "writeToken", 9090, true, func(res kmsg.Msg) {
				done <- res.E
			})
		})
	})
}
