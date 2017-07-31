package dht

import (
	"net"
	"os"

	"github.com/anacrolix/torrent/util"
	"github.com/mh-cbon/dht/security"
)

// Bootstrap your dht, find your closest peer for your node id in the network.
// if a recommended ip is returned, you should
// update the internal krpc publicip, re-generates an id, and bootstrap again.
func (d *DHT) Bootstrap(id []byte, publicIP *util.CompactPeer, addrs []string) (*util.CompactPeer, error) {
	return d.rpc.Boostrap(id, publicIP, addrs)
}

// BootstrapAuto handles bep42 bootstrap process.
func (d *DHT) BootstrapAuto(publicIP *util.CompactPeer, addrs []string) error {
	hostname, _ := os.Hostname()
	localAddr := d.rpc.Addr()
	for {
		var ip *net.IP
		if publicIP != nil {
			ip = &publicIP.IP
		}
		id := security.GenerateSecureNodeID(hostname, localAddr, ip)
		recommandedIP, err := d.rpc.Boostrap(id, publicIP, addrs)
		if err == nil && recommandedIP != nil {
			publicIP = recommandedIP
		} else {
			return err
		}
	}
}

// BootstrapExport the nodes in the bootstrap table.
func (d *DHT) BootstrapExport() (ret []string) {
	return d.rpc.BootstrapExport()
}
