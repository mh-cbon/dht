package rpc

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/anacrolix/torrent/iplist"
	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/socket"
)

func TestBootstrap(t *testing.T) {
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
	timeout := time.Millisecond * 10
	port := 9606
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
	makeRPC := func(name string, timeout time.Duration, opts ...KRPCOpt) *KRPC {
		opts = append([]KRPCOpt{
			KRPCOpts.ID(string(makID(name))),
			KRPCOpts.WithTimeout(timeout),
			KRPCOpts.WithAddr(newAddr()),
		}, opts...)
		return New(opts...)
	}
	t.Run("should error when there is no bootstrap", func(t *testing.T) {
		rpc := makeRPC("", timeout)
		_, err := rpc.Boostrap(nil, nil, []string{})
		wantErr(t, errors.New("nothing to resolve"), err)
	})
	t.Run("should error when the bootstrap node address does not exist", func(t *testing.T) {
		bob := makeRPC("", timeout)
		go bob.Listen(nil)
		defer bob.Close()
		_, err := bob.Boostrap(nil, nil, []string{"127.0.0.1:9568"})
		wantErr(t, errors.New("All bootstrap nodes failed [KRPC error 201: Query timeout]"), err)
	})
	t.Run("should error when the bootstrap node does not respond", func(t *testing.T) {
		alice := makeSocket("alice", timeout)
		go alice.MustListen(nil)
		defer alice.Close()

		bob := makeRPC("bob", timeout)
		go bob.Listen(nil)
		defer bob.Close()

		_, err := bob.Boostrap(nil, nil, []string{alice.GetAddr().String()})
		wantErr(t, errors.New("All bootstrap nodes failed [KRPC error 201: Query timeout]"), err)
	})
	t.Run("should bootstrap when the node respond", func(t *testing.T) {
		fred := makeSocket("fred", timeout)
		defer fred.Close()
		go fred.MustListen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			return fred.Respond(remote, msg.T, kmsg.Return{})
		})

		alice := makeSocket("alice", timeout)
		defer alice.Close()
		go alice.MustListen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			return alice.Respond(remote, msg.T, kmsg.Return{
				Nodes: compactNodes(NodeInfo(fred)),
			})
		})

		bob := makeRPC("bob", timeout)
		go bob.Listen(nil)
		defer bob.Close()

		_, err := bob.Boostrap(nil, nil, []string{alice.GetAddr().String()})
		rejectErr(t, err)

		table := bob.bootstrap
		if table.Count() != 1 {
			t.Errorf("table length incorrect wanted=%v got=%v", 1, table.Count())
		}
		got := table.Get(fred.GetID())
		if got == nil {
			t.Errorf("table content incorrect wanted=%v got=%v", "fred node", got)
		} else if bytes.Equal(got.GetID(), fred.GetID()) == false {
			t.Errorf("table content incorrect wanted=%v got=%v", fred.GetID(), got.GetID())
		} else if fred.GetAddr().String() != got.GetAddr().String() {
			t.Errorf("table content incorrect wanted=%v got=%v", fred.GetAddr().String(), got.GetAddr().String())
		}

		// target := strings.Repeat("a", 20)
		// done := make(chan bool)
		// res, err2 := rpc.Closest(target, "ping", nil, func(n Node, res kmsg.Msg) {
		// 	done <- true
		// })
		// rejectErr(t, err2)
		// if len(res) != 1 {
		// 	t.Errorf("Wanted len(res)=%v got=%v", 1, len(res))
		// }
		// if res[0].GetAddr().String() != fred.Addr().String() {
		// 	t.Errorf("Wanted res[0].GetAddr()=%v got=%v", fred.Addr().String(), res[0].GetAddr().String())
		// }
		// if string(res[0].GetID()) != fred.ID() {
		// 	t.Errorf("Wanted res[0].GetID()=%v got=%v", fred.ID(), string(res[0].GetID()))
		// }
		// select {
		// case <-done:
		// case <-time.After(time.Millisecond):
		// 	t.Errorf("Closest node not queried")
		// }
	})
	t.Run("should not bootstrap against bad nodes", func(t *testing.T) {
		alice := makeSocket("alice", timeout)
		defer alice.Close()
		go alice.MustListen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			return alice.Respond(remote, msg.T, kmsg.Return{})
		})

		bob := makeRPC("bob", timeout)
		go bob.Listen(nil)
		defer bob.Close()

		bob.BanNode(alice.GetAddr())
		_, err := bob.Boostrap(nil, nil, []string{alice.GetAddr().String()})
		wantErr(t, errors.New("The table is empty after bootstrap"), err)

		table := bob.bootstrap
		if table.Count() > 0 {
			t.Errorf("Table length incorrect wanted=%v got=%v", 0, table.Count())
		}
	})
	t.Run("should not bootstrap against excluded ips", func(t *testing.T) {
		alice := makeSocket("alice", timeout)
		defer alice.Close()
		go alice.MustListen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			return alice.Respond(remote, msg.T, kmsg.Return{})
		})

		ip := alice.GetAddr().IP
		ipExclude := newIPRange(net.IP(ip), net.IP(ip), "")

		bob := makeRPC("bob", timeout, KRPCOpts.BlockIPs(ipExclude))
		go bob.Listen(nil)
		defer bob.Close()

		_, err := bob.Boostrap(nil, nil, []string{alice.GetAddr().String()})
		wantErr(t, fmt.Errorf("All bootstrap nodes failed [IP blocked %v]", alice.GetAddr().String()), err)

		table := bob.bootstrap
		if table.Count() > 0 {
			t.Errorf("Table length incorrect wanted=%v got=%v", 0, table.Count())
		}
	})
	t.Run("should record returned nodes", func(t *testing.T) {

		var fredResponded bool
		var aliceResponded bool

		fred := makeSocket("fred", timeout)
		defer fred.Close()
		go fred.MustListen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			fredResponded = true
			return fred.Respond(remote, msg.T, kmsg.Return{})
		})

		alice := makeSocket("alice", timeout)
		defer alice.Close()
		go alice.MustListen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			aliceResponded = true
			return alice.Respond(remote, msg.T, kmsg.Return{
				ID:    string(makID("alice")),
				Nodes: compactNodes(NodeInfo(fred)),
			})
		})

		<-time.After(time.Millisecond)

		bob := makeRPC("bob", timeout)
		go bob.Listen(nil)
		defer bob.Close()

		_, err := bob.Boostrap(nil, nil, []string{alice.GetAddr().String()})
		rejectErr(t, err)

		table := bob.bootstrap
		if table.Count() != 1 {
			t.Errorf("table length incorrect wanted=%v got=%v", 1, table.Count())
		}
		got := table.Get(fred.GetID())
		if got == nil {
			t.Errorf("table content incorrect wanted=%v got=%v", "fred node", got)
		} else if bytes.Equal(got.GetID(), fred.GetID()) == false {
			t.Errorf("table content incorrect wanted=%v got=%v", fred.GetID(), got.GetID())
		} else if fred.GetAddr().String() != got.GetAddr().String() {
			t.Errorf("table content incorrect wanted=%v got=%v", fred.GetAddr().String(), got.GetAddr().String())
		}
		if !aliceResponded {
			t.Errorf("Wanted aliceResponded=%v got=%v", true, aliceResponded)
		}
		if !fredResponded {
			t.Errorf("Wanted fredResponded=%v got=%v", true, fredResponded)
		}
	})
}

func makID(in string) []byte {
	res := []byte(in)
	for i := len(res); i < 20; i++ {
		res = append(res, 0x0)
	}
	return res[:20]
}

func compactNodes(nodes ...kmsg.NodeInfo) kmsg.CompactIPv4NodeInfo {
	return kmsg.CompactIPv4NodeInfo(nodes)
}

func newIPRange(from, to net.IP, desc string) *iplist.IPList {
	return iplist.New(
		[]iplist.Range{
			iplist.Range{
				First:       from,
				Last:        to,
				Description: desc,
			},
		},
	)
}
