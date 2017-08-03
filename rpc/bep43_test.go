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
}
