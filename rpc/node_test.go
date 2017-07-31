package rpc

import (
	"bytes"
	"net"
	"testing"
)

func TestNewNode(t *testing.T) {

	t.Run("should copy the data when creating a new Node", func(t *testing.T) {
		id := [20]byte{0xf}
		addr := &net.UDPAddr{Port: 50}
		n := NewNode(id, addr)

		id[0] = 0x0
		addr.Port = 100

		if bytes.Equal(n.GetID(), id[:]) {
			t.Errorf("id not copied")
		}
		if addr.Port == n.GetAddr().Port {
			t.Errorf("addr not copied")
		}
	})
}
