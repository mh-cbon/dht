---
License: MIT
LicenseFile: LICENSE
LicenseColor: yellow
---
# {{.Name}}

{{template "badge/travis" .}} {{template "badge/appveyor" .}} {{template "badge/goreport" .}} {{template "badge/godoc" .}} {{template "license/shields" .}}

{{pkgdoc}}

It provides support for bep05 / bep42 / bep43 / bep44

Useful resources
- http://engineering.bittorrent.com/2013/01/22/bittorrent-tech-talks-dht/
- http://www.bittorrent.org/beps/bep_0000.html


# {{toc 5}}

# Api

#### > {{cat "example/main2.go" | color "go"}}

# Cli

the main.go file helps to play with the table.

```sh
  -annimplied
    	Announce an implied port.
  -annport int
    	The port of the announce query.
  -blocklist string
    	A peer guardian ip list to block (Some organization:1.0.0.0-1.255.255.255)
  -bnodes string
    	The boostrap node address list seprated by a coma, or public to use pre configured list of public bootstrap nodes, or 'no' to bootstrap empty
  -did string
    	The id of your node in the dht network
  -do string
    	The action to perform boostrap|closest_stores|closest_peers|ping|announce_peer|get_peers|find_node|get|put|genkey|sign (default "boostrap")
  -kconcurrency int
    	The number of nodes to refresh on k-bucket split (default 8)
  -keyname string
    	The name of the key
  -ksize int
    	The size of the k-buckets (default 20)
  -listen string
    	The listen address of the udp socket
  -pip string
    	The public ip of your node to share with other nodes.
  -qconcurrency int
    	The number of simultaneous outgoing requests (default 24)
  -qtimeout string
    	The query timeout (default "1s")
  -remote string
    	The remote address to query ip:port|closest_store|closest_peers
  -salt string
    	The salt to use to put/sign a value.
  -seq int
    	The seq value of get/put.
  -target string
    	Info hash target 20 bytes string hex encoded.
  -tsecret string
    	the secret of the token server
  -val string
    	A value to put.
  -vv
    	verbose mode
  -wait
    	Enter in wait mode after the command ran.
```

# Cli Examples

```sh
go run main.go -do closest_peers -target faf5c61ddcc5e8a2dabede0f3b482cd9aea9434c -qtimeout 1s -vv -bnodes public

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
