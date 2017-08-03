package socket

import (
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mh-cbon/dht/kmsg"
)

func TestCLimit(t *testing.T) {

	port := 9906
	newAddr := func() string {
		ip := "127.0.0.1"
		addr := fmt.Sprintf("%v:%v", ip, port)
		port++
		return addr
	}
	t.Run("should not exceed query limit", func(t *testing.T) {

		var tcnt int32
		var cnt int32
		concurrency := 5
		concurrencyMax := 14
		concurrencyMin := 2

		alice := New(RPCOpts.WithAddr(newAddr()))
		go alice.MustListen(func(msg kmsg.Msg, from *net.UDPAddr) error {
			<-time.After(time.Millisecond * 10)
			return alice.Respond(from, msg.T, kmsg.Return{V: "hi!"})
		})

		bob := NewConcurrent(concurrency, RPCOpts.WithAddr(newAddr()))
		go bob.MustListen(nil)

		<-time.After(time.Millisecond)

		done := make(chan bool)
		go func() {
			for {
				go atomic.AddInt32(&tcnt, 1)
				atomic.AddInt32(&cnt, 1)
				bob.Query(alice.GetAddr(), "meet", nil, func(kmsg.Msg) {
					atomic.AddInt32(&cnt, -1)
				})
				select {
				case <-time.After(time.Microsecond):
				case <-done:
					return
				}
			}
		}()
		go func() {
			<-time.After(time.Millisecond * 5)
			for {
				if cnt > int32(concurrencyMax) {
					t.Errorf("Exceeded concurrency max=%v wanted=%v got=%v", concurrencyMax, concurrency, cnt)
				} else if cnt < int32(concurrencyMin) {
					t.Errorf("Low concurrency min=%v wanted=%v got=%v", concurrencyMin, concurrency, cnt)
				}
				select {
				case <-time.After(time.Millisecond * 5):
				case <-done:
					return
				}
			}
		}()

		<-time.After(time.Millisecond * 250)
		if tcnt < 20 {
			t.Errorf("Unsufficent query executed wanted>%v got=%v", 50, tcnt)
		}
		if cnt > int32(concurrencyMax) {
			t.Errorf("Exceeded concurrency max=%v wanted=%v got=%v", concurrencyMax, concurrency, cnt)
		}
		done <- true

		bob.Close()
		alice.Close()
		<-time.After(time.Millisecond)
	})
	t.Run("should not exceed query limit2", func(t *testing.T) {
		alice := New(RPCOpts.WithAddr(newAddr()))
		i := 10
		go alice.MustListen(func(msg kmsg.Msg, from *net.UDPAddr) error {
			i--
			<-time.After(time.Duration(i) * time.Millisecond)
			return alice.Respond(from, msg.T, kmsg.Return{V: "hi!"})
		})

		bob := NewConcurrent(5, RPCOpts.WithAddr(newAddr()))
		go bob.MustListen(nil)

		<-time.After(time.Millisecond)

		go bob.Query(alice.GetAddr(), "meet", nil, func(kmsg.Msg) {
			log.Println("timeout 1")
		})
		go bob.Query(alice.GetAddr(), "meet", nil, func(kmsg.Msg) {
			log.Println("timeout 2")
		})
		go bob.Query(alice.GetAddr(), "meet", nil, func(kmsg.Msg) {
			log.Println("timeout 3")
		})

		<-time.After(time.Millisecond * 250)

		bob.Close()
		alice.Close()
	})

}
