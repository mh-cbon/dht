// to test repetitive multiple lookup and multiple table release.
package main

import (
	"flag"
	"fmt"
	"log"
	"sync"

	"github.com/mh-cbon/dht/bootstrap"
	"github.com/mh-cbon/dht/bucket"
	"github.com/mh-cbon/dht/dht"
	"github.com/mh-cbon/dht/logger"
)

// var shift = []byte{0x00, 0x00, 0x00, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
var shift = []byte{0x00, 0x00, 0x00, 0x0C, 0x0F, 0x0F, 0x0F, 0x0F, 0x0F, 0x0F, 0x0F, 0x0F, 0x0F, 0x0F, 0x0F, 0x0F, 0x0F, 0x0F, 0x0F, 0x0F}

//0x00 00 00 0C 0F0F0F0F 0F0F0F0F 0F0F0F0F 0F0F0F0F
func shitfId(id []byte, n int) [][]byte {
	ret := [][]byte{id}
	for i := 1; i < n; i++ {
		src := id
		if i > 0 {
			src = ret[i-1]
		}
		newTarget := make([]byte, len(src))
		for i := range src {
			if src[i] == 255 {
				newTarget[i] = 0
			} else {
				newTarget[i] = src[i] + 1
			}
		}
		ret = append(ret, newTarget)
	}
	return ret
}

func main3() {
	var v bool
	flag.BoolVar(&v, "vv", false, "verbose mode")
	flag.Parse()

	ready := func(d *dht.DHT) error {
		if v {
			d.AddLogger(logger.Text(log.Printf))
		}
		fmt.Println("Running bootstrap...")
		publicIP, err := d.BootstrapAuto(nil, bootstrap.Public)
		if err != nil {
			return err
		}
		if publicIP != nil {
			log.Printf("public IP bootstrap %v:%v\n", publicIP.IP, publicIP.Port)
		}

		selfID := []byte(d.ID())
		fmt.Printf("your node id %x\n", selfID)

		// that s good, we are ready.

		fmt.Println("Boostrap done...")

		// you can run a lookup to find nodes matching an info_hash or target.
		// info_hash and target are hex string.
		target := "caf5c61ddcc5e8a2dabede0f3b482cd9aea9434c"
		id, _ := dht.HexToBytes(target)
		for {
			ids := shitfId(id, 10)
			var wg sync.WaitGroup
			wg.Add(len(ids))
			log.Printf("Performing %v lookups\n", len(ids))
			for _, id := range ids {
				targetID := id
				var retErr error
				go func() {
					defer wg.Done()
					fmt.Printf("Performing lookup request for %x\n", targetID)
					todoTarget := fmt.Sprintf("%x", targetID)
					lookupErr := d.LookupStores(todoTarget, nil)
					if lookupErr != nil {
						retErr = lookupErr
						return
					}

					// then get the closest peers for that target
					closest, err := d.ClosestStores(todoTarget, 16)
					if err != nil {
						retErr = err
						return
					}
					log.Println()
					fmt.Printf("Found %v nodes close to %x\n", len(closest), targetID)
					for _, c := range closest {
						fmt.Printf("%-24v %x %v\n", c.GetAddr(), c.GetID(), bucket.Distance(targetID, c.GetID()))
					}
					if relErr := d.ReleaseLookupTable(todoTarget); relErr != nil {
						retErr = relErr
						return
					}
				}()
				if retErr != nil {
					return retErr
				}
			}
			wg.Wait()
			id = ids[len(ids)-1]
		}
		return nil
	}
	if err := dht.New(dht.DefaultOps()).Serve(ready); err != nil {
		if err != nil {
			log.Fatal(err)
		}
	}
}
