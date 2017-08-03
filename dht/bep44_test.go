package dht

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/rpc"
)

func TestBep44(t *testing.T) {
	timeout := time.Millisecond * 10
	port := 9106
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
	makeRPC := func(name string, timeout time.Duration) *rpc.KRPC {
		return rpc.New(
			rpc.KRPCOpts.ID(string(makID(name))),
			rpc.KRPCOpts.WithTimeout(timeout),
			rpc.KRPCOpts.WithAddr(newAddr()),
		)
	}
	makeDHT := func(name string, timeout time.Duration) *DHT {
		return New(
			Opts.ID(string(makID(name))),
			Opts.WithTimeout(timeout),
			Opts.WithAddr(newAddr()),
		)
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

	t.Run("should answer to get request", func(t *testing.T) {
		alice := makeRPC("alice", timeout)
		go alice.MustListen(nil)
		defer alice.Close()

		node := makeDHT("bob", timeout)
		defer node.Close()
		node.Serve(func(arg1 *DHT) error {
			w := make(chan bool)
			value := "Hello World!"
			h, goodTargetID, _ := ValueToHexAndByteString(value)
			if h != "e5f96f6f38320f0f33959cb4d3d656452117aadb" {
				t.Errorf("Inavvlid hex wanted=%v, got=%v", "e5f96f6f38320f0f33959cb4d3d656452117aadb", h)
			}
			alice.Get(node.GetAddr(), goodTargetID, func(msg kmsg.Msg) {
				rejectErrMsg(t, msg)
				rejectArgMsg(t, msg)
				wantRetID(t, msg, "bob\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
				if msg.R.Token == "" {
					t.Errorf("wanted msg.R.Token=%v, got=%v", "", msg.R.Token)
				} else if node.ValidateToken(msg.R.Token, alice.GetAddr()) == false {
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
	t.Run("should not answer to get request with bad target id", func(t *testing.T) {
		alice := makeRPC("alice", timeout)
		go alice.MustListen(nil)
		defer alice.Close()

		node := makeDHT("bob", timeout)
		defer node.Close()
		node.Serve(func(arg1 *DHT) error {
			w := make(chan bool)
			badTargetID := []byte("abcd")
			alice.Get(node.GetAddr(), badTargetID, func(msg kmsg.Msg) {
				rejectRetMsg(t, msg)
				rejectArgMsg(t, msg)
				wantErrCode(t, msg, kmsg.ErrorTimeout.Code)
				w <- true
			})
			<-w
			return nil
		})
	})

	t.Run("should answer to put request with valid token", func(t *testing.T) {
		alice := makeRPC("alice", timeout)
		go alice.MustListen(nil)
		defer alice.Close()

		node := makeDHT("bob", timeout)
		defer node.Close()
		node.Serve(func(arg1 *DHT) error {
			w := make(chan bool)
			value := "Hello World!"
			h, targetID, err := ValueToHexAndByteString(value)
			if err != nil {
				t.Error(err)
			}
			if h != "e5f96f6f38320f0f33959cb4d3d656452117aadb" {
				t.Errorf("Inavvlid hex wanted=%v, got=%v", "e5f96f6f38320f0f33959cb4d3d656452117aadb", h)
			}
			alice.Get(node.GetAddr(), targetID, func(msg kmsg.Msg) {
				rejectErrMsg(t, msg)
				rejectArgMsg(t, msg)
				wantRetID(t, msg, "bob\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
				if msg.R.Token == "" {
					t.Errorf("wanted msg.R.Token!=%q, got=%#v", "", msg.R.Token)
				}
				alice.Put(node.GetAddr(), value, msg.R.Token, func(msg kmsg.Msg) {
					rejectErrMsg(t, msg)
					rejectArgMsg(t, msg)
					wantRetID(t, msg, "bob\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")

					alice.Get(node.GetAddr(), targetID, func(msg kmsg.Msg) {
						rejectErrMsg(t, msg)
						rejectArgMsg(t, msg)
						wantRetID(t, msg, "bob\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
						if msg.R.V != value {
							t.Errorf("WAnted msg.R.V=%q, got=%q", value, msg.R.V)
						}
						w <- true
					})
				})
			})
			<-w
			return nil
		})
	})
	t.Run("should not answer to put request with bad value (too big)", func(t *testing.T) {
		alice := makeRPC("alice", timeout)
		go alice.MustListen(nil)
		defer alice.Close()

		node := makeDHT("bob", timeout)
		defer node.Close()
		node.Serve(func(arg1 *DHT) error {
			w := make(chan bool)
			value := strings.Repeat("a", 1001)
			alice.Put(node.GetAddr(), value, "", func(msg kmsg.Msg) {
				rejectRetMsg(t, msg)
				rejectArgMsg(t, msg)
				wantErrCode(t, msg, kmsg.ErrorTimeout.Code)
				w <- true
			})
			<-w
			return nil
		})
	})
	t.Run("should not answer to put request with bad value (too small)", func(t *testing.T) {
		alice := makeRPC("alice", timeout)
		go alice.MustListen(nil)
		defer alice.Close()

		node := makeDHT("bob", timeout)
		defer node.Close()
		node.Serve(func(arg1 *DHT) error {
			w := make(chan bool)
			alice.Put(node.GetAddr(), "", "", func(msg kmsg.Msg) {
				rejectRetMsg(t, msg)
				rejectArgMsg(t, msg)
				wantErrCode(t, msg, kmsg.ErrorTimeout.Code)
				w <- true
			})
			<-w
			return nil
		})
	})
	t.Run("should not answer to put request with empty token", func(t *testing.T) {
		alice := makeRPC("alice", timeout)
		go alice.MustListen(nil)
		defer alice.Close()

		node := makeDHT("bob", timeout)
		defer node.Close()
		node.Serve(func(arg1 *DHT) error {
			w := make(chan bool)
			value := strings.Repeat("a", 8)
			alice.Put(node.GetAddr(), value, "", func(msg kmsg.Msg) {
				rejectRetMsg(t, msg)
				rejectArgMsg(t, msg)
				wantErrCode(t, msg, kmsg.ErrorTimeout.Code)
				w <- true
			})
			<-w
			return nil
		})
	})
	t.Run("should respond error to put request with wrong token", func(t *testing.T) {
		alice := makeRPC("alice", timeout)
		go alice.MustListen(nil)
		defer alice.Close()

		node := makeDHT("bob", timeout)
		defer node.Close()
		node.Serve(func(arg1 *DHT) error {
			w := make(chan bool)
			writeToken := "nop"
			value := strings.Repeat("a", 8)
			alice.Put(node.GetAddr(), value, writeToken, func(msg kmsg.Msg) {
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
