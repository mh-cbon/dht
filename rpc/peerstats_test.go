package rpc

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/socket"
)

func TestPeerStats(t *testing.T) {
	timeout := time.Millisecond * 10
	port := 9686
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
	t.Run("should ban node if it responds with 3 timeouts", func(t *testing.T) {
		bob := makeSocket("bob", "127.0.0.1", timeout)
		bobRPC := New(bob, KRPCConfig{})
		go bobRPC.Listen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			if msg.RO != 1 {
				t.Errorf("wanted alice to add ro flag=1, got=%v", msg.RO)
			}
			return bob.Respond(remote, msg.T, kmsg.Return{V: "hello"})
		})
		defer bobRPC.Close()

		ad := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port + 1}
		addrs := []*net.UDPAddr{ad, ad, ad, ad, ad}
		bobRPC.BatchAddrs(addrs, func(addr *net.UDPAddr, done chan<- error) (*socket.Tx, error) {
			return bobRPC.Query(addr, "ping", nil, func(res kmsg.Msg) {
				done <- res.E
			})
		})
		<-time.After(timeout * 2)
		if bobRPC.isBadNode(ad) == false {
			t.Errorf("wanted bob to ban %q", ad.String())
		}
	})
	t.Run("should ban node if it is ro", func(t *testing.T) {
		bob := makeSocket("bob", "127.0.0.1", timeout)
		bobRPC := New(bob, KRPCConfig{})
		go bobRPC.Listen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			if msg.RO != 1 {
				t.Errorf("wanted alice to add ro flag=1, got=%v", msg.RO)
			}
			return bob.Respond(remote, msg.T, kmsg.Return{V: "hello"})
		})
		defer bobRPC.Close()

		alice := makeSocket("alice", "127.0.0.1", timeout)
		go alice.MustListen(nil)
		defer alice.Close()
		alice.ReadOnly(true)
		alice.Query(bob.Addr(), "ping", nil, func(res kmsg.Msg) {})
		<-time.After(timeout * 2)
		if bobRPC.isBadNode(alice.Addr()) == false {
			t.Errorf("wanted bob to ban alice (%q)", alice.Addr().String())
		}
	})
	t.Run("should unban node if it is not ro anymore", func(t *testing.T) {
		bob := makeSocket("bob", "127.0.0.1", timeout)
		bobRPC := New(bob, KRPCConfig{})
		go bobRPC.Listen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			return bob.Respond(remote, msg.T, kmsg.Return{V: "hello"})
		})
		defer bobRPC.Close()

		alice := makeSocket("alice", "127.0.0.1", timeout)
		go alice.MustListen(nil)
		defer alice.Close()
		alice.ReadOnly(true)
		alice.Query(bob.Addr(), "ping", nil, func(res kmsg.Msg) {})
		<-time.After(timeout * 2)
		if bobRPC.isBadNode(alice.Addr()) == false {
			t.Errorf("wanted bob to ban alice (%q)", alice.Addr().String())
		}
		alice.ReadOnly(false)
		alice.Query(bob.Addr(), "ping", nil, func(res kmsg.Msg) {})
		<-time.After(timeout * 2)
		if bobRPC.isBadNode(alice.Addr()) {
			t.Errorf("wanted bob to unban alice (%q)", alice.Addr().String())
		}
	})
	t.Run("should unban node if it responds", func(t *testing.T) {
		bob := makeSocket("bob", "127.0.0.1", timeout)
		bobRPC := New(bob, KRPCConfig{})
		go bobRPC.Listen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			if msg.RO != 1 {
				t.Errorf("wanted alice to add ro flag=1, got=%v", msg.RO)
			}
			return bob.Respond(remote, msg.T, kmsg.Return{V: "hello"})
		})
		defer bobRPC.Close()

		ad := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port + 1}
		addrs := []*net.UDPAddr{ad, ad, ad, ad, ad}
		bobRPC.BatchAddrs(addrs, func(addr *net.UDPAddr, done chan<- error) (*socket.Tx, error) {
			return bobRPC.Query(addr, "ping", nil, func(res kmsg.Msg) {
				done <- res.E
			})
		})
		<-time.After(timeout * 2)
		if bobRPC.isBadNode(ad) == false {
			t.Errorf("wanted bob to record %q as bad node", ad.String())
		}

		alice := makeSocket("alice", "127.0.0.1", timeout)
		go alice.MustListen(nil)
		defer alice.Close()

		addrs = []*net.UDPAddr{alice.Addr()}
		bobRPC.BatchAddrs(addrs, func(addr *net.UDPAddr, done chan<- error) (*socket.Tx, error) {
			return bobRPC.Query(addr, "ping", nil, func(res kmsg.Msg) {
				done <- res.E
			})
		})
		<-time.After(timeout * 2)
		if bobRPC.isBadNode(alice.Addr()) {
			t.Errorf("wanted bob to unban alice (%q)", alice.Addr().String())
		}
	})
}
