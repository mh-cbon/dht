// Package dht is a kademlia implementation of a Distributed Hash Table.
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/anacrolix/torrent/iplist"
	"github.com/anacrolix/torrent/util"
	"github.com/mh-cbon/dht/bootstrap"
	"github.com/mh-cbon/dht/bucket"
	"github.com/mh-cbon/dht/crypto"
	"github.com/mh-cbon/dht/dht"
	"github.com/mh-cbon/dht/ed25519"
	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/logger"
	"github.com/mh-cbon/dht/rpc"
	"github.com/mh-cbon/dht/security"
	"github.com/mh-cbon/dht/socket"
)

func safeXORBytes(dst, a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		dst[i] = a[i] ^ b[i]
	}
	return n
}

func main() {

	var tokenSecret string
	var qconcurreny int
	var qtimeout string
	var kconcurrency int
	var ksize int
	var dID string
	var sListen string
	var sPublicIP string
	var do string
	var remotes string
	var blocklist string
	var targetHash string
	var putVal string
	var putSalt string
	var announcePort int
	var announceImpliedPort bool
	var bnodesList string
	var keyName string
	var seq int
	var v bool
	var wait bool
	flag.StringVar(&tokenSecret, "tsecret", "", "the secret of the token server")
	flag.IntVar(&qconcurreny, "qconcurrency", 24, "The number of simultaneous outgoing requests")
	flag.StringVar(&qtimeout, "qtimeout", "1s", "The query timeout")
	flag.IntVar(&kconcurrency, "kconcurrency", 8, "The number of nodes to refresh on k-bucket split")
	flag.IntVar(&ksize, "ksize", 20, "The size of the k-buckets")
	flag.StringVar(&dID, "did", "", "The id of your node in the dht network")
	flag.StringVar(&sListen, "listen", "", "The listen address of the udp socket")
	flag.StringVar(&sPublicIP, "pip", "", "The public ip of your node to share with other nodes.")
	flag.StringVar(&bnodesList, "bnodes", "", "The boostrap node address list seprated by a coma, or public to use pre configured list of public bootstrap nodes, or 'no' to bootstrap empty")
	flag.StringVar(&do, "do", "boostrap", "The action to perform boostrap|closest_stores|closest_peers|ping|announce_peer|get_peers|find_node|get|put|genkey|sign")
	flag.StringVar(&remotes, "remote", "", "The remote address to query ip:port|closest_store|closest_peers")
	flag.StringVar(&blocklist, "blocklist", "", "A peer guardian ip list to block (Some organization:1.0.0.0-1.255.255.255)")
	flag.StringVar(&targetHash, "target", "", "Info hash target 20 bytes string hex encoded.")
	flag.StringVar(&putVal, "val", "", "A value to put.")
	flag.StringVar(&putSalt, "salt", "", "The salt to use to put/sign a value.")
	flag.IntVar(&seq, "seq", 0, "The seq value of get/put.")
	flag.IntVar(&announcePort, "annport", 0, "The port of the announce query.")
	flag.BoolVar(&announceImpliedPort, "annimplied", false, "Announce an implied port.")
	flag.BoolVar(&v, "vv", false, "verbose mode")
	flag.BoolVar(&wait, "wait", false, "Enter in wait mode after the command ran.")
	flag.StringVar(&keyName, "keyname", "", "The name of the key")

	flag.Parse()

	if do == "genkey" {
		if keyName == "" {
			panic("required -keyname <argument>")
		}
		pvk, pbk, err := ed25519.PvkFromDir(".", keyName)
		if err != nil {
			panic(err)
		}
		fmt.Printf("private key (raw length:%3v) %x\n", len(pvk), pvk)
		fmt.Printf("public key  (raw length:%3v) %x\n", len(pbk), pbk)
		return
	}

	if do == "sign" {
		if keyName == "" {
			panic("required -keyname <argument>")
		}
		pvk, pbk, err := ed25519.PvkFromDir(".", keyName)
		if err != nil {
			panic(err)
		}
		encodedValue, err := rpc.EncodePutValue(putVal, putSalt, seq)
		if err != nil {
			panic(err)
		}
		sign := ed25519.Sign(pvk, pbk, []byte(encodedValue))
		valueHash := crypto.HashSha1(string(pbk), putSalt)
		fmt.Printf("benc val    (length:%3v)     %v\n", len(encodedValue), encodedValue)
		fmt.Printf("private key (raw length:%3v) %x\n", len(pvk), pvk)
		fmt.Printf("public key  (raw length:%3v) %x\n", len(pbk), pbk)
		fmt.Printf("value sign  (raw length:%3v) %x\n", len(sign), sign)
		fmt.Printf("target id   (raw length:%3v) %v\n", len(valueHash), valueHash)
		return
	}

	var publicIP *util.CompactPeer
	var bNodes []string

	hostname, _ := os.Hostname()

	if sPublicIP != "" {
		host, sport, err := net.SplitHostPort(sPublicIP)
		if err != nil {
			panic(err)
		}
		port, err := strconv.Atoi(sport)
		if err != nil {
			panic(err)
		}
		publicIP = &util.CompactPeer{IP: net.ParseIP(host), Port: port}
	}

	ready := func(public *dht.DHT) error {

		selfID := public.GetID()
		log.Printf("self id %x\n", public.ID())

		log.Println("dht bootstrap...")
		log.Println("bootstrap nodes count:", len(bNodes))

		recommendedIP := publicIP
		if rIP, bootErr := public.Bootstrap(selfID, publicIP, bNodes); bootErr != nil && len(bNodes) > 0 {
			return bootErr
		} else if rIP != nil {
			recommendedIP = rIP
			selfID = security.GenerateSecureNodeID(hostname, public.GetAddr(), &rIP.IP)
			log.Printf("After bootstrap a new recommended ip was provided %v\n", rIP)
			if rIP2, bootErr2 := public.Bootstrap(selfID, rIP, bNodes); bootErr != nil {
				return bootErr2
			} else if rIP2 != nil {
				log.Printf("After bootstrap a new recommended ip was provided %v\n", rIP2)
				return fmt.Errorf("Stopping now as something is going wrong.")
			}
		}
		if len(bNodes) > 0 {
			bNodes = saveBNodes(recommendedIP, public)
			log.Println("bootstrap length:", len(bNodes))
		}
		log.Printf("self id after bootstrap %x\n", public.ID())
		if recommendedIP != nil {
			log.Printf("public IP bootstrap %v:%v\n", recommendedIP.IP, recommendedIP.Port)
		}

		if do == "boostrap" {
			return nil
		}

		var target []byte
		if targetHash == "self" {
			target = make([]byte, len(selfID))
			copy(target, selfID)
			targetHash = fmt.Sprintf("%x", target)

		} else if len(targetHash) > 0 {
			if len(targetHash) < 20 {
				targetHash += strings.Repeat("0", 20-len(targetHash))
			}
			t, e := hex.DecodeString(targetHash)
			if e != nil {
				return e
			}
			target = t
		}
		log.Printf("target %x\n", target)
		log.Printf("target hash %v\n", targetHash)

		if do == "closest_stores" {
			err := public.LookupStores(targetHash, nil)
			if err != nil {
				return err
			}
			contacts, err := public.ClosestStores(targetHash, 16)
			if err != nil {
				return err
			}
			log.Println("closest store length:", len(contacts))
			for _, c := range contacts {
				log.Printf("%-24v %x %v\n", c.GetAddr(), c.GetID(), bucket.Distance(target, c.GetID()))
			}
			return nil

		} else if do == "closest_peers" {
			err := public.LookupPeers(targetHash, nil)
			if err != nil {
				return err
			}
			contacts, err := public.ClosestPeers(targetHash, 16)
			if err != nil {
				return err
			}
			log.Println("closest peer length:", len(contacts))
			for _, c := range contacts {
				log.Printf("%-24v %x %vn", c.GetAddr(), c.GetID(), bucket.Distance(c.GetID(), target))
			}
			return nil
		}

		var targetAddr []*net.UDPAddr
		if remotes == "closest_store" {
			contacts, err := public.ClosestStores(targetHash, 8)
			if err != nil {
				return err
			}
			targetAddr = []*net.UDPAddr{}
			for _, c := range contacts {
				targetAddr = append(targetAddr, c.GetAddr())
			}
		} else if remotes == "closest_peers" {
			contacts, err := public.ClosestPeers(targetHash, 8)
			if err != nil {
				return err
			}
			targetAddr = []*net.UDPAddr{}
			for _, c := range contacts {
				targetAddr = append(targetAddr, c.GetAddr())
			}
		} else if remotes != "" {
			var err error
			targetAddr, err = bootstrap.Addrs(strings.Split(remotes, ","))
			if err != nil {
				return err
			}
		}

		onResponse := func(remote *net.UDPAddr, res kmsg.Msg) {
			log.Println(remote, res.R, res.E)
		}
		if do == "ping" {
			public.PingAll(targetAddr, onResponse)

		} else if do == "announce_peer" {
			public.AnnouncePeerAll(targetAddr, targetHash, uint(announcePort), announceImpliedPort, onResponse)

		} else if do == "get_peers" {
			public.GetPeersAll(targetAddr, targetHash, onResponse)

		} else if do == "find_node" {
			public.FindNodeAll(targetAddr, targetHash, onResponse)

		} else if do == "get" {

			if keyName != "" {
				_, pbk, err := ed25519.PvkFromDir(".", keyName)
				if err != nil {
					return err
				}
				log.Printf("pbk %x\n", pbk)
				err = public.LookupStores(targetHash, nil)
				if err != nil {
					return err
				}
				val, err := public.MGetAll(targetHash, pbk, seq, putSalt, targetAddr...)
				log.Println("value", val)
				return err
			}
			if err := public.LookupStores(targetHash, nil); err != nil {
				return err
			}
			val, err := public.GetAll(targetHash, targetAddr...)
			log.Println("value", val)
			return err

		} else if do == "put" {

			if keyName != "" {
				pvk, _, err := ed25519.PvkFromDir(".", keyName)
				if err != nil {
					return err
				}
				m, err := dht.PutFromPvk(putVal, putSalt, pvk, seq, 0)
				if err != nil {
					return err
				}
				log.Printf("target %v\n", m.Target)
				log.Printf("sign %x\n", m.Sign)
				log.Printf("pbk %x\n", m.Pbk)
				err = public.LookupStores(m.Target, nil)
				if err != nil {
					return err
				}
				return public.MPutAll(m, targetAddr...)
			}
			hexTarget := dht.ValueToHex(putVal)
			log.Println("target", hexTarget)
			err := public.LookupStores(hexTarget, nil)
			if err != nil {
				return err
			}
			_, err = public.PutAll(putVal, targetAddr...)
			return err
		}
		return nil
	}

	opts := []dht.Opt{}

	socket := socket.NewConcurrent(qconcurreny)
	if v {
		socket.AddLogger(logger.Text(log.Printf))
	}
	opts = append(opts, dht.Opts.WithRPCSocket(socket))
	opts = append(opts, dht.Opts.WithAddr(sListen))

	if tokenSecret != "" {
		opts = append(opts, dht.Opts.WithTokenSecret([]byte(tokenSecret)))
	}

	if qtimeout != "" {
		d, err := time.ParseDuration(qtimeout)
		if err != nil {
			panic(err)
		}
		opts = append(opts, dht.Opts.WithTimeout(d))
	}
	if dID != "" {
		if len(dID) < 20 {
			dID += strings.Repeat("0", 20-len(dID))
		}
		t, e := hex.DecodeString(dID)
		if e != nil {
			panic(e)
		}
		opts = append(opts, dht.Opts.ID(string(t)))
	} else {
		var pip *net.IP
		if publicIP != nil {
			pip = &publicIP.IP
		}
		i := security.GenerateSecureNodeID(hostname, nil, pip)
		opts = append(opts, dht.Opts.ID(string(i)))
	}

	// var ipBlocked *iplist.IPList
	if blocklist != "" {
		i, err := iplist.NewFromReader(strings.NewReader(blocklist))
		if err != nil {
			panic(err)
		}
		// ipBlocked = i
		opts = append(opts, dht.Opts.BlockIPs(i))
	}

	opts = append(opts, dht.Opts.WithConcurrency(kconcurrency))
	opts = append(opts, dht.Opts.WithK(ksize))

	if bnodesList != "no" {
		publicIP, bNodes = loadBNodes(bnodesList)
	}

	node := dht.New(opts...)
	if err := node.ListenAndServe(dht.StdQueryHandler(node), ready); err != nil {
		if bnodesList == "no" {
			if _, ok := err.(emptyBootstrap); ok {
				err = nil
			}
		}
		if err != nil {
			log.Fatal(err)
		}
	}
	if wait {
		log.Println("entering wait mode, hit ctrl+c to quit")
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		close(c)
	}
}

type emptyBootstrap interface {
	IsEmptyBootstrap() bool
}

var bootstrapFile = "bootstrap.json"

func saveBNodes(publicIP *util.CompactPeer, public *dht.DHT) []string {
	bNodes := public.BootstrapExport()
	if len(bNodes) > 0 {
		if err := bootstrap.Save(bootstrapFile, publicIP, bNodes); err != nil {
			panic(err)
		}
	}
	return bNodes
}
func loadBNodes(bnodesList string) (oldIP *util.CompactPeer, nodes []string) {
	x, err := bootstrap.Get(bootstrapFile)
	if err == nil {
		oldIP = x.OldIP
		nodes = x.Nodes
	}
	if bnodesList == "public" {
		nodes = bootstrap.Public
	} else if bnodesList != "" {
		nodes = strings.Split(bnodesList, ",")
		for i, n := range nodes {
			nodes[i] = strings.TrimSpace(n)
		}
	}
	return
}
