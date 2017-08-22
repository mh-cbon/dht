// Package example provides some demonstration of this api.
package main

// q&d speed test.

import (
	"flag"
	"log"
	"sync"

	"github.com/mh-cbon/dht/dht"
	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/rpc"
)

func main1() {

	flag.Parse()

	ready := func(d *dht.DHT) error {
		alice := rpc.New(
			rpc.KRPCOpts.WithAddr("127.0.0.1:9569"),
			rpc.KRPCOpts.ID("alice"),
		)
		go alice.Listen(nil)
		remote := d.GetAddr()
		id := [20]byte{}
		e := 10000
		var wg sync.WaitGroup
		wg.Add(e)
		for i := 0; i < e; i++ {
			alice.GetPeers(remote, id[:], func(res kmsg.Msg) {
				if res.E != nil {
					log.Println(res.E)
				}
				wg.Done()
			})
		}
		wg.Wait()
		return nil
	}
	if err := dht.New(dht.DefaultOps()).Serve(ready); err != nil {
		if err != nil {
			log.Fatal(err)
		}
	}
}
