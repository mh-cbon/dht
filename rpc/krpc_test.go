package rpc

import (
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/socket"
)

func TestKRPC(t *testing.T) {
	timeout := time.Millisecond * 10
	wantErr := func(t *testing.T, wanted error, got error) {
		if got == nil {
			t.Errorf("Wanted err=%v, got err=%v", wanted, got)
			t.FailNow()
			return
		}
		if wanted.Error() != got.Error() {
			t.Errorf("Wanted err=%v, got err=%v", wanted.Error(), got.Error())
			t.FailNow()
			return
		}
	}
	rejectErr := func(t *testing.T, got error) {
		if got != nil {
			t.Errorf("Wanted err=%v, got err=%v", nil, got)
			t.FailNow()
			return
		}
	}
	port := 9706
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
	t.Run("should query/respond", func(t *testing.T) {
		alice := makeSocket("alice", timeout)
		go alice.MustListen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			return alice.Respond(remote, msg.T, kmsg.Return{V: "hello"})
		})
		defer alice.Close()

		bob := makeRPC("bob", timeout)
		go bob.Listen(nil)
		defer bob.Close()

		_, err := bob.Query(alice.GetAddr(), "ping", nil, func(res kmsg.Msg) {
			if res.R.V != "hello" {
				t.Errorf("invalid response, wanted V=%v, got=%v", "hello", res.R.V)
			}
			rejectErr(t, res.E)
		})
		rejectErr(t, err)
	})
	t.Run("should query/error", func(t *testing.T) {
		alice := makeSocket("alice", timeout)
		go alice.MustListen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			return alice.Error(remote, msg.T, kmsg.Error{Code: 404})
		})
		defer alice.Close()

		bob := makeRPC("bob", timeout)
		go bob.Listen(nil)
		defer bob.Close()

		_, err := bob.Query(alice.GetAddr(), "ping", nil, func(res kmsg.Msg) {
			if res.E.Code != 404 {
				t.Errorf("invalid response, wanted V=%v, got=%v", "hello", res.E.Code)
			}
			rejectErr(t, res.E)
		})
		rejectErr(t, err)
	})
	t.Run("should query/respond with timeout", func(t *testing.T) {
		bob := makeRPC("bob", timeout*10)
		go bob.Listen(nil)
		defer bob.Close()

		addr := &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 6556,
		}
		now := time.Now()
		_, err := bob.Query(addr, "ping", nil, func(res kmsg.Msg) {
			wantErr(t, fmt.Errorf("KRPC error 201: Query timeout"), res.E)
			d := time.Now().Sub(now)
			if d < 100*time.Millisecond || d > 120*time.Millisecond {
				t.Errorf("the timeout duration is incorrect, wanted 100<%v<110", d)
			}
		})
		rejectErr(t, err)
		<-time.After(timeout * 14)
	})
	t.Run("should call timeout callback", func(t *testing.T) {
		bob := makeRPC("bob", timeout)
		go bob.Listen(nil)
		defer bob.Close()

		addr := &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 6556,
		}

		bob.GetPeersStats().OnPeerTimeout("test", func(remote *net.UDPAddr, queriedQ string, queriedA map[string]interface{}, response kmsg.Msg) {
			if remote.String() != addr.String() {
				t.Errorf("wanted q=%v, got=%v", addr.String(), remote.String())
			}
			if queriedQ != "ping" {
				t.Errorf("wanted q=%v, got=%v", "ping", queriedQ)
			}
			if queriedA != nil {
				t.Errorf("wanted a=%v, got=%v", nil, queriedA)
			}
			if response.E.Code != 201 {
				t.Errorf("wanted e.Code=%v, got=%v", 201, response.E.Code)
			}
		})

		_, err := bob.Query(addr, "ping", nil, func(res kmsg.Msg) {})
		rejectErr(t, err)
		<-time.After(timeout * 14)
	})
	t.Run("should batch", func(t *testing.T) {
		alice := makeSocket("alice", timeout)
		go alice.Listen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			return alice.Respond(remote, msg.T, kmsg.Return{V: string(alice.GetID())})
		})
		fred := makeSocket("fred", timeout)
		go fred.Listen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			return fred.Respond(remote, msg.T, kmsg.Return{V: string(fred.GetID())})
		})

		bob := makeRPC("bob", timeout)
		go bob.Listen(nil)
		defer bob.Close()

		ids := []string{string(alice.GetID()), string(fred.GetID())}
		addrs := []*net.UDPAddr{alice.GetAddr(), fred.GetAddr()}
		errs := bob.Batch(len(addrs), func(i int, done chan<- error) (*socket.Tx, error) {
			addr := addrs[i]
			id := ids[i]
			return bob.Query(addr, "ping", nil, func(res kmsg.Msg) {
				if res.R.V != string(id) {
					t.Errorf("Incorrect value received, wanted=%v, got=%v", string(id), res.R.V)
				}
				done <- res.E
			})
		})
		if len(errs) != 0 {
			t.Errorf("wanted len(errs)=0, got errs=%v", errs)
		}
		<-time.After(timeout * 2)
	})
	t.Run("should batch in //", func(t *testing.T) {

		bob := makeRPC("bob", timeout)
		go bob.Listen(nil)
		defer bob.Close()

		addrs := []*net.UDPAddr{
			&net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port + 1},
			&net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port + 2},
			&net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port + 3},
			&net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port + 4},
		}
		now := time.Now()
		errs := bob.Batch(len(addrs), func(i int, done chan<- error) (*socket.Tx, error) {
			return bob.Query(addrs[i], "ping", nil, func(res kmsg.Msg) {
				done <- res.E
			})
		})
		d := time.Now().Sub(now)
		min := timeout
		max := timeout + (time.Millisecond * 2)
		if d < min || d > max {
			t.Errorf("the timeout duration is incorrect, wanted %v<%v<%v", min, d, max)
		}
		if len(errs) == 0 {
			t.Errorf("wanted len(errs)>0, got errs=%v", errs)
		}
		<-time.After(timeout * 2)
	})
	t.Run("should return query errors when batching", func(t *testing.T) {

		bob := makeRPC("bob", timeout)
		go bob.Listen(nil)
		defer bob.Close()

		errs := bob.Batch(2, func(i int, done chan<- error) (*socket.Tx, error) {
			return nil, errors.New("nop")
		})
		if len(errs) != 2 {
			t.Errorf("wanted len(errs)=2, got errs=%v", errs)
		} else {
			e := errs[0]
			if e.Error() != "nop" {
				t.Errorf("wanted e.Error()=%v, got=%v", "nop", e.Error())
			}
		}
		<-time.After(timeout * 2)
	})
	t.Run("should return response errors when batching", func(t *testing.T) {
		alice := makeSocket("alice", timeout)
		go alice.Listen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			return alice.Error(remote, msg.T, kmsg.Error{Code: 666, Msg: "alice"})
		})
		fred := makeSocket("fred", timeout)
		go fred.Listen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			return fred.Error(remote, msg.T, kmsg.Error{Code: 666, Msg: "fred"})
		})

		bob := makeRPC("bob", timeout)
		go bob.Listen(nil)
		defer bob.Close()

		addrs := []*net.UDPAddr{alice.GetAddr(), fred.GetAddr()}
		errs := bob.Batch(len(addrs), func(i int, done chan<- error) (*socket.Tx, error) {
			return bob.Query(addrs[i], "ping", nil, func(res kmsg.Msg) {
				done <- res.E
			})
		})
		if len(errs) != 2 {
			t.Errorf("wanted len(errs)=2, got errs=%v", errs)
		} else {
			e, ok := errs[0].(*kmsg.Error)
			if !ok {
				t.Errorf("wanted errs[0].(*krpc.Error)=true, got=%v", false)
			} else if e.Code != 666 {
				t.Errorf("wanted errs[0].Code=%v, got=%v", 666, e.Code)
			}
		}
		<-time.After(timeout * 2)
	})
}
