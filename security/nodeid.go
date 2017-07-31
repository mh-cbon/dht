package security

import (
	"crypto"
	"net"

	"github.com/anacrolix/missinggo"
)

// GenerateSecureNodeID generates an secure node id for given details.
// hostname or empty.
// localAddr of the socket.
// publicIP or nil.
func GenerateSecureNodeID(hostname string, localAddr net.Addr, publicIP *net.IP) []byte {
	var id [20]byte
	h := crypto.SHA1.New()
	ss := hostname + localAddr.String()
	h.Write([]byte(ss))
	if b := h.Sum(id[:0:20]); len(b) != 20 {
		panic(len(b))
	}
	if len(id) != 20 {
		panic(len(id))
	}
	if publicIP == nil {
		k := missinggo.AddrIP(localAddr)
		publicIP = &k
	}
	SecureNodeID(id[:], *publicIP)
	return id[:]
}
