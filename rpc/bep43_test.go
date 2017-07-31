package rpc

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/socket"
)

func TestBep43(t *testing.T) {
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
	t.Run("should ban node if it sends a query with read-only flag", func(t *testing.T) {

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
			t.Error("wanted bob to record alice as bad node")
		}
	})
}
