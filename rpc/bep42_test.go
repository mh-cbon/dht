package rpc

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/socket"
)

func TestBep42(t *testing.T) {
	timeout := time.Millisecond * 10
	port := 9676
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
	t.Run("should blank a response get token when the node id is insecure", func(t *testing.T) {
		remote := &net.UDPAddr{
			IP:   net.ParseIP("90.215.23.3"),
			Port: 6666,
		}
		SecuredResponseOnly(remote, func(res kmsg.Msg) {
			if res.R.Token != "" {
				t.Errorf("wanted res.R.Token=%v, got=%v", "", res.R.Token)
			}
		})(kmsg.Msg{R: &kmsg.Return{Token: "to blank", ID: string(makID("insecure"))}})
	})
	t.Run("should block insecure queries", func(t *testing.T) {

		bob := makeSocket("bob", "127.0.0.1", timeout)
		go bob.Listen(nil)
		defer bob.Close()
		bobRPC := New(bob, KRPCConfig{})

		err := SecuredQueryOnly(bobRPC, func(msg kmsg.Msg, remote *net.UDPAddr) error {
			return nil
		})(
			kmsg.Msg{Q: kmsg.QGetPeers},
			&net.UDPAddr{IP: net.ParseIP("95.12.45.2"), Port: 8888},
		)
		wantErr(t, fmt.Errorf("Invalid get_peers packet: mising Arguments"), err)

		err2 := SecuredQueryOnly(bobRPC, func(msg kmsg.Msg, remote *net.UDPAddr) error {
			return nil
		})(
			kmsg.Msg{Q: kmsg.QGetPeers, A: &kmsg.MsgArgs{ID: string(bob.GetID())}},
			&net.UDPAddr{IP: net.ParseIP("95.12.45.2"), Port: 8888},
		)
		if err2 == nil {
			t.Errorf("wanted err!=nil, got=%v", err2)
		}
		<-time.After(timeout * 2)
	})
}
