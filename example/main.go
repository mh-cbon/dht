// to test speed...
package main

import (
	"flag"
	"log"
	"sync"

	"github.com/mh-cbon/dht/dht"
	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/rpc"
	"github.com/mh-cbon/dht/socket"
)

func main1() {

	flag.Parse()

	var secret []byte

	sockConfig := socket.RPCConfig{}.WithAddr("127.0.0.1:9568").WithID([]byte("bob"))

	q := 8

	sock := socket.NewConcurrent(q, sockConfig)
	// sock.AddLogger(logger.Text(log.Printf))
	kconfig := rpc.KRPCConfig{}.WithConcurrency(8).WithK(20)
	bob := rpc.New(sock, kconfig)

	listening := func(d *dht.DHT) error {
		sockConfig2 := socket.RPCConfig{}.WithAddr("127.0.0.1:9569").WithID([]byte("alice"))
		sock2 := socket.NewConcurrent(q, sockConfig2)
		// sock2.AddLogger(logger.Text(log.Printf))
		alice := rpc.New(sock2, kconfig)
		go alice.Listen(nil)
		remote := bob.Addr()
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
	if err := dht.New(secret, bob).Listen(listening); err != nil {
		if err != nil {
			log.Fatal(err)
		}
	}
}
