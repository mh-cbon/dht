package socket

import (
	"net"
	"testing"
	"time"

	"github.com/mh-cbon/dht/kmsg"
)

func TestRPC(t *testing.T) {
	t.Run("query + response", func(t *testing.T) {
		done := make(chan bool)

		alice := New(NewConfig("127.0.0.1:9526"))
		go alice.MustListen(func(msg kmsg.Msg, from *net.UDPAddr) error {
			wanted := "hello!"
			if msg.A.V != wanted {
				t.Errorf("alice wanted bob message=%v got=%v", wanted, msg.A.V)
			}
			return alice.Respond(from, msg.T, kmsg.Return{V: "hi!"})
		})

		bob := New(NewConfig("127.0.0.1:9527"))
		go bob.MustListen(nil)

		<-time.After(time.Millisecond)
		a := map[string]interface{}{"v": "hello!"}
		bob.Query(makeAddr("127.0.0.1:9526"), "meet", a, func(res kmsg.Msg) {
			wanted := "hi!"
			if res.R.V != wanted {
				t.Errorf("bob wanted alice response=%v got=%v", wanted, res.R.V)
			}
			done <- true
		})

		<-done
		alice.Close()
		bob.Close()
	})

	t.Run("parallel query", func(t *testing.T) {
		alice := New(NewConfig("127.0.0.1:9528"))
		go alice.MustListen(func(msg kmsg.Msg, from *net.UDPAddr) error {
			return alice.Respond(from, msg.T, kmsg.Return{V: msg.A.V})
		})

		bob := New(NewConfig("127.0.0.1:9529"))
		go bob.MustListen(nil)

		<-time.After(time.Millisecond)
		done := make(chan bool)
		a := map[string]interface{}{"v": "echo1"}
		go bob.Query(makeAddr("127.0.0.1:9528"), "meet", a, func(res kmsg.Msg) {
			wanted := "echo1"
			if res.R.V != wanted {
				t.Errorf("bob wanted alice response=%v got=%v", wanted, res.R.V)
			}
			done <- true
		})

		b := map[string]interface{}{"v": "echo2"}
		go bob.Query(makeAddr("127.0.0.1:9528"), "meet", b, func(res kmsg.Msg) {
			wanted := "echo2"
			if res.R == nil {
				t.Errorf("bob wanted alice response=%v got=%v", wanted, res.R)
			} else if res.R.V != wanted {
				t.Errorf("bob wanted alice response=%v got=%v", wanted, res.R.V)
			}
			done <- true
		})

		<-done
		<-done
		alice.Close()
		bob.Close()
	})

	t.Run("query + error", func(t *testing.T) {
		alice := New(NewConfig("127.0.0.1:9530"))
		go alice.MustListen(func(msg kmsg.Msg, from *net.UDPAddr) error {
			return alice.Error(from, msg.T, kmsg.Error{Code: 10, Msg: "plop"})
		})

		bob := New(NewConfig("127.0.0.1:9531"))
		go bob.MustListen(nil)

		<-time.After(time.Millisecond)
		done := make(chan bool)
		a := map[string]interface{}{"v": "echo1"}
		go bob.Query(makeAddr("127.0.0.1:9530"), "meet", a, func(res kmsg.Msg) {
			wanted := "plop"
			wantedCode := 10
			if res.E == nil {
				t.Errorf("bob wanted error=not nil got=%v", res.E)
			} else if res.E.Msg != wanted {
				t.Errorf("bob wanted alice respond msg=%v got=%v", wanted, res.E.Msg)
			} else if res.E.Code != wantedCode {
				t.Errorf("bob wanted alice respond code=%v got=%v", wanted, res.E.Code)
			}
			done <- true
		})

		<-done
		alice.Close()
		bob.Close()
	})

	t.Run("query timeout", func(t *testing.T) {
		var aliceGotMsg bool
		alice := New(NewConfig("127.0.0.1:9532"))
		go alice.MustListen(func(msg kmsg.Msg, from *net.UDPAddr) error {
			aliceGotMsg = true
			<-time.After(time.Second)
			return alice.Error(from, msg.T, kmsg.Error{Code: 10, Msg: "plop"})
		})

		bob := New(NewConfig("127.0.0.1:9533").WithTimeout(time.Millisecond * 10))
		go bob.MustListen(nil)

		<-time.After(time.Millisecond)
		done := make(chan bool)
		a := map[string]interface{}{"v": "echo1"}
		go bob.Query(makeAddr("127.0.0.1:9532"), "meet", a, func(res kmsg.Msg) {
			wanted := "Query timeout"
			wantedCode := 201
			if res.E == nil {
				t.Errorf("bob wanted error=not nil got=%v", res.E)
			} else if res.E.Msg != wanted {
				t.Errorf("bob wanted alice respond msg=%v got=%v", wanted, res.E.Msg)
			} else if res.E.Code != wantedCode {
				t.Errorf("bob wanted alice respond code=%v got=%v", wanted, res.E.Code)
			}
			done <- true
		})

		<-done
		if !aliceGotMsg {
			t.Errorf("alice got message wanted=%v got=%v", true, false)
		}
		alice.Close()
		bob.Close()
	})
}

func makeAddr(addr string) *net.UDPAddr {
	ret, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		panic(err)
	}
	return ret
}
