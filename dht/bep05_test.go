package dht

import (
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/rpc"
	"github.com/mh-cbon/dht/socket"
)

func makID(in string) []byte {
	res := []byte(in)
	for i := len(res); i < 20; i++ {
		res = append(res, 0x0)
	}
	return res[:20]
}

func TestBep05(t *testing.T) {
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
	makeRPC := func(name string, ip string, timeout time.Duration) *rpc.KRPC {
		soc := makeSocket(name, ip, timeout)
		return rpc.New(soc, rpc.KRPCConfig{})
	}
	rejectErrMsg := func(t *testing.T, msg kmsg.Msg) {
		if msg.E != nil {
			t.Errorf("wanted msg.E==nil, got=%v", msg.E)
		}
	}
	rejectArgMsg := func(t *testing.T, msg kmsg.Msg) {
		if msg.A != nil {
			t.Errorf("wanted msg.A==nil, got=%v", msg.A)
		}
	}
	rejectRetMsg := func(t *testing.T, msg kmsg.Msg) {
		if msg.R != nil {
			t.Errorf("wanted msg.R==nil, got=%v", msg.R)
		}
	}
	wantErrCode := func(t *testing.T, msg kmsg.Msg, code int) {
		if msg.E == nil {
			t.Errorf("wanted msg.E!=nil, got=%v", nil)
		} else if msg.E.Code != code {
			t.Errorf("wanted msg.E.Code=%v, got=%#v", code, msg.E.Code)
		}
	}
	wantRetID := func(t *testing.T, msg kmsg.Msg, id string) {
		if msg.R == nil {
			t.Errorf("wanted msg.R!=nil, got=%v", nil)
		} else if msg.R.ID != id {
			t.Errorf("wanted msg.R.Id=%v, got=%#v", "bob", msg.R.ID)
		}
	}
	t.Run("should answer to find_node request", func(t *testing.T) {
		alice := makeRPC("alice", "127.0.0.1", timeout)
		go alice.MustListen(nil)
		defer alice.Close()

		bob := makeRPC("bob", "127.0.0.1", timeout)
		node := New(nil, bob)
		defer node.Close()
		node.Listen(func(arg1 *DHT) error {
			w := make(chan bool)
			goodTargetID := []byte(strings.Repeat("a", 20))
			alice.FindNode(bob.Addr(), goodTargetID, func(msg kmsg.Msg) {
				rejectErrMsg(t, msg)
				rejectArgMsg(t, msg)
				wantRetID(t, msg, "bob\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
				//todo: add testing of nodes field
				w <- true
			})
			<-w
			return nil
		})
	})
	t.Run("should not answer to find_node request with bad target id", func(t *testing.T) {
		alice := makeRPC("alice", "127.0.0.1", timeout)
		go alice.MustListen(nil)
		defer alice.Close()

		bob := makeRPC("bob", "127.0.0.1", timeout)
		node := New(nil, bob)
		defer node.Close()
		node.Listen(func(arg1 *DHT) error {
			w := make(chan bool)
			badTargetID := []byte("abcd")
			alice.FindNode(bob.Addr(), badTargetID, func(msg kmsg.Msg) {
				rejectRetMsg(t, msg)
				rejectArgMsg(t, msg)
				wantErrCode(t, msg, kmsg.ErrorTimeout.Code)
				w <- true
			})
			<-w
			return nil
		})
	})

	t.Run("should answer to ping request", func(t *testing.T) {
		alice := makeRPC("alice", "127.0.0.1", timeout)
		go alice.MustListen(nil)
		defer alice.Close()

		bob := makeRPC("bob", "127.0.0.1", timeout)
		node := New(nil, bob)
		defer node.Close()
		node.Listen(func(arg1 *DHT) error {
			w := make(chan bool)
			alice.Ping(bob.Addr(), func(msg kmsg.Msg) {
				rejectErrMsg(t, msg)
				rejectArgMsg(t, msg)
				wantRetID(t, msg, "bob\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
				w <- true
			})
			<-w
			return nil
		})
	})

	t.Run("should answer to get_peers request", func(t *testing.T) {
		alice := makeRPC("alice", "127.0.0.1", timeout)
		go alice.MustListen(nil)
		defer alice.Close()

		bob := makeRPC("bob", "127.0.0.1", timeout)
		node := New(nil, bob)
		defer node.Close()
		node.Listen(func(arg1 *DHT) error {
			w := make(chan bool)
			goodTargetID := []byte(strings.Repeat("a", 20))
			alice.GetPeers(bob.Addr(), goodTargetID, func(msg kmsg.Msg) {
				rejectErrMsg(t, msg)
				rejectArgMsg(t, msg)
				wantRetID(t, msg, "bob\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
				if msg.R.Token == "" {
					t.Errorf("wanted msg.R.Token=%v, got=%v", "", msg.R.Token)
				} else if node.ValidateToken(msg.R.Token, alice.Addr()) == false {
					t.Errorf("wanted valid token, got=%v", false)
				}
				//todo: add testing of peers field
				//todo: add testing of nodes field
				w <- true
			})
			<-w
			return nil
		})
	})
	t.Run("should not answer to get_peers request with bad target id", func(t *testing.T) {
		alice := makeRPC("alice", "127.0.0.1", timeout)
		go alice.MustListen(nil)
		defer alice.Close()

		bob := makeRPC("bob", "127.0.0.1", timeout)
		node := New(nil, bob)
		defer node.Close()
		node.Listen(func(arg1 *DHT) error {
			w := make(chan bool)
			badTargetID := []byte("abcd")
			alice.GetPeers(bob.Addr(), badTargetID, func(msg kmsg.Msg) {
				rejectRetMsg(t, msg)
				rejectArgMsg(t, msg)
				wantErrCode(t, msg, kmsg.ErrorTimeout.Code)
				w <- true
			})
			<-w
			return nil
		})
	})

	t.Run("should answer to announce_peer request with valid token", func(t *testing.T) {
		alice := makeRPC("alice", "127.0.0.1", timeout)
		go alice.MustListen(nil)
		defer alice.Close()

		bob := makeRPC("bob", "127.0.0.1", timeout)
		node := New(nil, bob)
		defer node.Close()
		node.Listen(func(arg1 *DHT) error {
			w := make(chan bool)
			goodTargetID := []byte(strings.Repeat("a", 20))
			alice.GetPeers(bob.Addr(), goodTargetID, func(msg kmsg.Msg) {
				rejectErrMsg(t, msg)
				rejectArgMsg(t, msg)
				wantRetID(t, msg, "bob\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
				if msg.R.Token == "" {
					t.Errorf("wanted msg.R.Token!=%q, got=%#v", "", msg.R.Token)
				}
				alice.AnnouncePeer(bob.Addr(), goodTargetID, msg.R.Token, 40, false, func(msg kmsg.Msg) {
					rejectErrMsg(t, msg)
					rejectArgMsg(t, msg)
					wantRetID(t, msg, "bob\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")

					alice.GetPeers(bob.Addr(), goodTargetID, func(msg kmsg.Msg) {
						rejectErrMsg(t, msg)
						rejectArgMsg(t, msg)
						wantRetID(t, msg, "bob\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
						log.Println("msg.E", msg.E)
						log.Println("msg.R", msg.R.Values)
						if len(msg.R.Values) == 0 {
							t.Errorf("Wanted len(msg.R.Values)>0, got=%v", len(msg.R.Values))
						} else {
							c := msg.R.Values[0]
							w := fmt.Sprintf("%v:%v", alice.Addr().IP, 40) // becasue implied port = false
							g := fmt.Sprintf("%v:%v", c.IP, c.Port)
							if w != g {
								t.Errorf("wanted msg.R.Values=%v, got=%v", w, g)
							}
						}
						w <- true
					})
				})
			})
			<-w
			return nil
		})
	})
	t.Run("should not answer to announce_peer request with bad target id", func(t *testing.T) {
		alice := makeRPC("alice", "127.0.0.1", timeout)
		go alice.MustListen(nil)
		defer alice.Close()

		bob := makeRPC("bob", "127.0.0.1", timeout)
		node := New(nil, bob)
		defer node.Close()
		node.Listen(func(arg1 *DHT) error {
			w := make(chan bool)
			badTargetID := []byte("abcd")
			alice.AnnouncePeer(bob.Addr(), badTargetID, "", 40, true, func(msg kmsg.Msg) {
				rejectRetMsg(t, msg)
				rejectArgMsg(t, msg)
				wantErrCode(t, msg, kmsg.ErrorTimeout.Code)
				w <- true
			})
			<-w
			return nil
		})
	})
	t.Run("should not answer to announce_peer request with empty token", func(t *testing.T) {
		alice := makeRPC("alice", "127.0.0.1", timeout)
		go alice.MustListen(nil)
		defer alice.Close()

		bob := makeRPC("bob", "127.0.0.1", timeout)
		node := New(nil, bob)
		defer node.Close()
		node.Listen(func(arg1 *DHT) error {
			w := make(chan bool)
			goodTargetID := []byte(strings.Repeat("a", 20))
			alice.AnnouncePeer(bob.Addr(), goodTargetID, "", 40, true, func(msg kmsg.Msg) {
				rejectRetMsg(t, msg)
				rejectArgMsg(t, msg)
				wantErrCode(t, msg, kmsg.ErrorTimeout.Code)
				w <- true
			})
			<-w
			return nil
		})
	})
	t.Run("should respond error to announce_peer request with wrong token", func(t *testing.T) {
		alice := makeRPC("alice", "127.0.0.1", timeout)
		go alice.MustListen(nil)
		defer alice.Close()

		bob := makeRPC("bob", "127.0.0.1", timeout)
		node := New(nil, bob)
		defer node.Close()
		node.Listen(func(arg1 *DHT) error {
			w := make(chan bool)
			goodTargetID := []byte(strings.Repeat("a", 20))
			writeToken := "nop"
			alice.AnnouncePeer(bob.Addr(), goodTargetID, writeToken, 40, true, func(msg kmsg.Msg) {
				rejectRetMsg(t, msg)
				rejectArgMsg(t, msg)
				wantErrCode(t, msg, kmsg.ErrorBadToken.Code)
				w <- true
			})
			<-w
			return nil
		})
	})
}
