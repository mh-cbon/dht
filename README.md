# DHT - kademlia

This is a golang implementation of a kadmelia Districbuted Hash Table.

It provides support for bep05 / bep42 / bep43 / bep44

Useful resources
- http://engineering.bittorrent.com/2013/01/22/bittorrent-tech-talks-dht/
- http://www.bittorrent.org/beps/bep_0000.html

# Api

```go
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
			d.SetLog(logger.Std)
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
		fmt.Printf("Performing lookup request for%x\n", targetID)
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
		return nil
	}
	if err := dht.New(nil, nil).Listen(listen); err != nil {
		if err != nil {
			log.Fatal(err)
		}
	}
}
```

# Cli Examples

```sh
go run main.go -do closest_peers -target faf5c61ddcc5e8a2dabede0f3b482cd9aea9434c -qtimeout 1s -vv

# run a node on 127...:9091, with an empty bootstrap, verbose, wait mode
go run main.go -listen 127.0.0.1:9091 -wait -bnodes no -vv

# sign a value prints useful information like keys, target etc
go run main.go -do sign -keyname bep44test -val "Hello World!"
go run main.go -do sign -keyname bep44test -val "Hello World!" -salt foobar

# put a value on a specific address (127.0.0.1:9090)
go run main.go -listen 127.0.0.1:9091 -bnodes no -remote "127.0.0.1:9090" -do put -val "Hello World!" -vv
# get a value from a remote
go run main.go -listen 127.0.0.1:9091 -bnodes no -remote "127.0.0.1:9090" -do get -target e5f96f6f38320f0f33959cb4d3d656452117aadb -vv

# put a mutable value on a specific address
go run main.go -listen 127.0.0.1:9091 -bnodes no -remote "127.0.0.1:9090" -do put -keyname bep44test -val "Hello World!" -salt foobar -seq 1 -vv
# get a mutable value on a specific address
go run main.go -listen 127.0.0.1:9091 -bnodes no -remote "127.0.0.1:9090" -do get -keyname bep44test -target 411eba73b6f087ca51a3795d9c8c938d365e32c1 -salt foobar -seq 1 -vv


```
