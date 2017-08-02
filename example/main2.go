// to test repetitive single lookup.
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/mh-cbon/dht/bootstrap"
	"github.com/mh-cbon/dht/bucket"
	"github.com/mh-cbon/dht/dht"
	"github.com/mh-cbon/dht/logger"
)

func main() {
	var v bool
	flag.BoolVar(&v, "vv", false, "verbose mode")
	flag.Parse()

	listen := func(d *dht.DHT) error {
		if v {
			d.AddLogger(logger.Text(log.Printf))
		}
		fmt.Println("Running bootstrap...")
		err := d.BootstrapAuto(nil, bootstrap.Public)
		if err != nil {
			return err
		}

		selfID := []byte(d.ID())
		fmt.Printf("your node id %x\n", selfID)

		// that s good, we are ready.

		fmt.Println("Boostrap done...")

		// you can run a lookup to find nodes matching an info_hash or target.
		// info_hash and target are hex string.
		target := "faf5c61ddcc5e8a2dabede0f3b482cd9aea9434c"
		targetID, _ := dht.HexToBytes(target)
		for {
			fmt.Printf("Performing lookup request for %x\n", targetID)
			lookupErr := d.LookupStores(target, nil)
			if lookupErr != nil {
				return lookupErr
			}

			// then get the closest peers for that target
			closest, err := d.ClosestStores(target, 16)
			if err != nil {
				return err
			}
			fmt.Printf("Found %v nodes close to %x\n", len(closest), targetID)
			for _, c := range closest {
				fmt.Printf("%-24v %x %v\n", c.GetAddr(), c.GetID(), bucket.Distance(targetID, c.GetID()))
			}
		}
		return nil
	}
	if err := dht.New(nil, nil).Listen(listen); err != nil {
		if err != nil {
			log.Fatal(err)
		}
	}
}
