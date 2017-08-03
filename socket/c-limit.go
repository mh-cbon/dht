package socket

import (
	"io"
	"net"

	"github.com/mh-cbon/dht/kmsg"
)

// Concurrent constraint the number of outgoing queries.
type Concurrent struct {
	*RPC
	qStart chan func(chan struct{})
	qDone  chan struct{}
	stop   chan struct{}
}

// NewConcurrent prepares a new socket with concurrency limit.
func NewConcurrent(limit int, opts ...RPCOpt) *Concurrent {
	ret := &Concurrent{
		RPC:    New(opts...),
		stop:   make(chan struct{}),
		qStart: make(chan func(chan struct{}), limit),
		qDone:  make(chan struct{}, limit),
	}
	return ret
}

// Listen reads kmsg.Msg
func (k *Concurrent) Listen(h QueryHandler) error {
	go k.Start()
	return k.RPC.Listen(h)
}

// MustListen reads kmsg.Msg
func (k *Concurrent) MustListen(h QueryHandler) {
	if err := k.Listen(h); err != nil && err != io.EOF {
		panic(err)
	}
}

// Close the socket.
func (k *Concurrent) Close() error {
	k.stop <- struct{}{}
	return k.RPC.Close()
}

// Start concurrently runs c simultaneous queries.
func (k *Concurrent) Start() {
	for {
		select {
		case f := <-k.qStart:
			f(k.qDone)
			k.qDone <- struct{}{}
		case <-k.stop:
			return
		}
	}
}

// Query qeueus a new query and wait for its completion.
func (k *Concurrent) Query(addr *net.UDPAddr, q string, a map[string]interface{}, onResponse func(kmsg.Msg)) (tx *Tx, err error) {
	k.qStart <- func(done chan struct{}) {
		k.RPC.Query(addr, q, a, func(m kmsg.Msg) {
			if onResponse != nil {
				go onResponse(m)
			}
			<-done
		})
	}
	return
}
