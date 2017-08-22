package socket

import (
	"fmt"
	"net"

	"github.com/anacrolix/torrent/bencode"
	"github.com/mh-cbon/dht/kmsg"
)

//Bencoded packets reader/writer
type Bencoded struct {
	*Server
}

// Listen reads bencoded packets
func (s Bencoded) Listen(read func(kmsg.Msg, *net.UDPAddr) error) error {
	return s.Server.Listen(func(b []byte, addr *net.UDPAddr) error {
		if len(b) < 2 || b[0] != 'd' || b[len(b)-1] != 'e' {
			return fmt.Errorf("Invalid bencoded packet %v", string(b))
		}
		var d kmsg.Msg
		err := bencode.Unmarshal(b, &d)
		if err != nil {
			return err
		}
		if read != nil {
			return read(d, addr)
		}
		return nil
	})
}

// Write bencoded packets.
func (s Bencoded) Write(m map[string]interface{}, node *net.UDPAddr) (err error) {
	b, err := bencode.Marshal(m)
	if err != nil {
		return
	}
	return s.Server.Write(b, node)
}
