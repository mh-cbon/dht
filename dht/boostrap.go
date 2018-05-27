package dht

import (
	"net"
	"os"

	"github.com/mh-cbon/dht/util"
	"github.com/mh-cbon/dht/security"
)

// Bootstrap an initial dht table. Such table defines your location in the network and your ID's neighbors.
// bep42: If a recommanded ip is returned, you should consider saving that ip as your public ip,
// derive a new id, and restart the bootstrap.
// if an error is returned, the table should not be considere as correctly built.
func (d *DHT) Bootstrap(id []byte, publicIP *util.CompactPeer, initialAddrs []string) (*util.CompactPeer, error) {
	return d.rpc.Boostrap(id, publicIP, initialAddrs)
}

// BootstrapAuto will automatically derive an ID for given publicIP
// and restart the bootstrap until your public ip is confirmed by the network.
func (d *DHT) BootstrapAuto(publicIP *util.CompactPeer, initialAddrs []string) (*util.CompactPeer, error) {
	hostname, _ := os.Hostname()
	localAddr := d.rpc.GetAddr()
	for {
		var ip *net.IP
		if publicIP != nil {
			ip = &publicIP.IP
		}
		id := security.GenerateSecureNodeID(hostname, localAddr, ip)
		recommandedIP, err := d.rpc.Boostrap(id, publicIP, initialAddrs)
		if err == nil && recommandedIP != nil {
			publicIP = recommandedIP
		} else if err != nil {
			return nil, err
		} else if publicIP != nil {
			break
		}
	}
	return publicIP, nil
}

// BootstrapExport addresses in the bootstrap table.
func (d *DHT) BootstrapExport() (ret []string) {
	return d.rpc.BootstrapExport()
}
