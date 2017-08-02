package rpc

import (
	"bytes"
	"errors"
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
	t.Run("should error when there is no bootstrap", func(t *testing.T) {
		sock := socket.New(socket.RPCConfig{}.WithTimeout(timeout))
		rpc := New(sock, KRPCConfig{})
		_, err := rpc.Boostrap(nil, nil, []string{})
		wantErr(t, errors.New("nothing to resolve"), err)
	})
	t.Run("should error when the bootstrap node address does not exist", func(t *testing.T) {
		sock := socket.New(socket.RPCConfig{}.WithTimeout(timeout))
		rpc := New(sock, KRPCConfig{})
		_, err := rpc.Boostrap(nil, nil, []string{"127.0.0.1:9568"})
		wantErr(t, errors.New("All bootstrap nodes failed [KRPC error 201: Query timeout]"), err)
	})
	t.Run("should error when the bootstrap node does not respond", func(t *testing.T) {
		alice := socket.New(socket.RPCConfig{}.WithAddr("127.0.0.1:9556"))
		go alice.MustListen(nil)
		defer alice.Close()

		bob := socket.New(socket.RPCConfig{}.WithTimeout(timeout))
		go bob.Listen(nil)
		defer bob.Close()

		rpc := New(bob, KRPCConfig{})
		_, err := rpc.Boostrap(nil, nil, []string{alice.Addr().String()})
		wantErr(t, errors.New("All bootstrap nodes failed [KRPC error 201: Query timeout]"), err)
	})
	t.Run("should bootstrap when the node respond", func(t *testing.T) {
		fred := socket.New(socket.RPCConfig{}.WithID(makID("fred")).WithAddr("127.0.0.1:9560"))
		defer fred.Close()
		go fred.MustListen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			return fred.Respond(remote, msg.T, kmsg.Return{})
		})

		alice := socket.New(socket.RPCConfig{}.WithID(makID("alice")).WithAddr("127.0.0.1:9558"))
		defer alice.Close()
		go alice.MustListen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			return alice.Respond(remote, msg.T, kmsg.Return{
				Nodes: compactNodes(NodeInfo(fred)),
			})
		})

		bob := socket.New(socket.RPCConfig{}.WithID(makID("bob")).WithTimeout(timeout))
		go bob.Listen(nil)
		defer bob.Close()

		rpc := New(bob, KRPCConfig{})
		_, err := rpc.Boostrap(nil, nil, []string{alice.Addr().String()})
		rejectErr(t, err)

		table := rpc.bootstrap
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
		alice := socket.New(socket.RPCConfig{}.WithAddr("127.0.0.1:9557"))
		defer alice.Close()
		go alice.MustListen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			return alice.Respond(remote, msg.T, kmsg.Return{})
		})

		bob := socket.New(socket.RPCConfig{}.WithTimeout(timeout))
		go bob.Listen(nil)
		defer bob.Close()

		rpc := New(bob, KRPCConfig{})
		rpc.BanNode(alice.Addr())
		_, err := rpc.Boostrap(nil, nil, []string{alice.Addr().String()})
		wantErr(t, errors.New("The table is empty after bootstrap"), err)

		table := rpc.bootstrap
		if table.Count() > 0 {
			t.Errorf("Table length incorrect wanted=%v got=%v", 0, table.Count())
		}
	})
	t.Run("should not bootstrap against excluded ips", func(t *testing.T) {
		alice := socket.New(socket.RPCConfig{}.WithAddr("127.0.0.1:9558"))
		defer alice.Close()
		go alice.MustListen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			return alice.Respond(remote, msg.T, kmsg.Return{})
		})

		ip := alice.Addr().IP
		ipExclude := newIPRange(net.IP(ip), net.IP(ip), "")

		bob := socket.New(socket.RPCConfig{}.WithIPBlockList(ipExclude).WithTimeout(timeout))
		go bob.Listen(nil)
		defer bob.Close()

		rpc := New(bob, KRPCConfig{})
		_, err := rpc.Boostrap(nil, nil, []string{alice.Addr().String()})
		wantErr(t, errors.New("All bootstrap nodes failed [IP blocked 127.0.0.1:9558]"), err)

		table := rpc.bootstrap
		if table.Count() > 0 {
			t.Errorf("Table length incorrect wanted=%v got=%v", 0, table.Count())
		}
	})
	t.Run("should record returned nodes", func(t *testing.T) {

		var fredResponded bool
		var aliceResponded bool

		fred := socket.New(socket.RPCConfig{}.WithID(makID("fred")).WithAddr("127.0.0.1:9560"))
		defer fred.Close()
		go fred.MustListen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			fredResponded = true
			return fred.Respond(remote, msg.T, kmsg.Return{})
		})

		alice := socket.New(socket.RPCConfig{}.WithID(makID("alice")).WithAddr("127.0.0.1:9561"))
		defer alice.Close()
		go alice.MustListen(func(msg kmsg.Msg, remote *net.UDPAddr) error {
			aliceResponded = true
			return alice.Respond(remote, msg.T, kmsg.Return{
				ID:    string(makID("alice")),
				Nodes: compactNodes(NodeInfo(fred)),
			})
		})

		<-time.After(time.Millisecond)
		bob := socket.New(socket.RPCConfig{}.WithTimeout(timeout))
		go bob.Listen(nil)
		defer bob.Close()

		rpc := New(bob, KRPCConfig{})
		_, err := rpc.Boostrap(nil, nil, []string{alice.Addr().String()})
		rejectErr(t, err)

		table := rpc.bootstrap
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
