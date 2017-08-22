package rpc

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/socket"
)

//todo : move to socket package.
func TestBep43(t *testing.T) {
	timeout := time.Millisecond * 10
	port := 9406
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
	t.Run("should ban node if it sends a query with read-only flag", func(t *testing.T) {

		bob := makeRPC("bob", timeout)
		go bob.Listen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			if msg.RO != 1 {
				t.Errorf("wanted alice to add ro flag=1, got=%v", msg.RO)
			}
			return bob.Respond(remote, msg.T, kmsg.Return{V: "hello"})
		})
		defer bob.Close()

		alice := makeSocket("alice", timeout)

		alice.ReadOnly(true)
		alice.Query(bob.GetAddr(), "ping", nil, func(res kmsg.Msg) {})
		<-time.After(timeout * 2)
		if bob.IsBadNode(alice.GetAddr()) == false {
			t.Errorf("wanted bob to record alice (%q) as bad node", alice.GetAddr().String())
		}
	})
	t.Run("should ban node if it responds with 3 timeouts", func(t *testing.T) {
		bob := makeRPC("bob", timeout)
		go bob.Listen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			if msg.RO != 1 {
				t.Errorf("wanted alice to add ro flag=1, got=%v", msg.RO)
			}
			return bob.Respond(remote, msg.T, kmsg.Return{V: "hello"})
		})
		defer bob.Close()

		ad := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port + 1}
		addrs := []*net.UDPAddr{ad, ad, ad, ad, ad}
		bob.BatchAddrs(addrs, func(addr *net.UDPAddr, done chan<- error) (*socket.Tx, error) {
			return bob.Query(addr, "ping", nil, func(res kmsg.Msg) {
				done <- res.E
			})
		})
		<-time.After(timeout * 2)
		if bob.IsBadNode(ad) == false {
			t.Errorf("wanted bob to ban %q", ad.String())
		}
	})
	t.Run("should ban node if it is ro", func(t *testing.T) {
		bob := makeRPC("bob", timeout)
		go bob.Listen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			if msg.RO != 1 {
				t.Errorf("wanted alice to add ro flag=1, got=%v", msg.RO)
			}
			return bob.Respond(remote, msg.T, kmsg.Return{V: "hello"})
		})
		defer bob.Close()

		alice := makeSocket("alice", timeout)
		go alice.MustListen(nil)
		defer alice.Close()

		alice.ReadOnly(true)
		alice.Query(bob.GetAddr(), "ping", nil, func(res kmsg.Msg) {})
		<-time.After(timeout * 2)
		if bob.IsBadNode(alice.GetAddr()) == false {
			t.Errorf("wanted bob to ban alice (%q)", alice.GetAddr().String())
		}
	})
	t.Run("should unban node if it is not ro anymore", func(t *testing.T) {
		bob := makeRPC("bob", timeout)
		go bob.Listen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			return bob.Respond(remote, msg.T, kmsg.Return{V: "hello"})
		})
		defer bob.Close()

		alice := makeSocket("alice", timeout)
		go alice.MustListen(nil)
		defer alice.Close()

		alice.ReadOnly(true)
		alice.Query(bob.GetAddr(), "ping", nil, func(res kmsg.Msg) {})
		<-time.After(timeout * 2)
		if bob.IsBadNode(alice.GetAddr()) == false {
			t.Errorf("wanted bob to ban alice (%q)", alice.GetAddr().String())
		}
		alice.ReadOnly(false)
		alice.Query(bob.GetAddr(), "ping", nil, func(res kmsg.Msg) {})
		<-time.After(timeout * 2)
		if bob.IsBadNode(alice.GetAddr()) {
			t.Errorf("wanted bob to unban alice (%q)", alice.GetAddr().String())
		}
	})
	t.Run("should unban node if it responds", func(t *testing.T) {
		bob := makeRPC("bob", timeout)
		go bob.Listen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			if msg.RO != 1 {
				t.Errorf("wanted alice to add ro flag=1, got=%v", msg.RO)
			}
			return bob.Respond(remote, msg.T, kmsg.Return{V: "hello"})
		})
		defer bob.Close()

		ad := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port + 1}
		addrs := []*net.UDPAddr{ad, ad, ad, ad, ad}
		bob.BatchAddrs(addrs, func(addr *net.UDPAddr, done chan<- error) (*socket.Tx, error) {
			return bob.Query(addr, "ping", nil, func(res kmsg.Msg) {
				done <- res.E
			})
		})
		<-time.After(timeout * 2)
		if bob.IsBadNode(ad) == false {
			t.Errorf("wanted bob to record %q as bad node", ad.String())
		}

		alice := makeSocket("alice", timeout)
		go alice.MustListen(nil)
		defer alice.Close()

		addrs = []*net.UDPAddr{alice.GetAddr()}
		bob.BatchAddrs(addrs, func(addr *net.UDPAddr, done chan<- error) (*socket.Tx, error) {
			return bob.Query(addr, "ping", nil, func(res kmsg.Msg) {
				done <- res.E
			})
		})
		<-time.After(timeout * 2)
		if bob.IsBadNode(alice.GetAddr()) {
			t.Errorf("wanted bob to unban alice (%q)", alice.GetAddr().String())
		}
	})
}
