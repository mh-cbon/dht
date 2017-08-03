package socket

import (
	"fmt"
	"io"
	"log"
	"net"
)

//ServerOpt is a option setter
type ServerOpt func(*Server)

// ServerOpts are net server options.
var ServerOpts = struct {
	WithSocket func(socket net.PacketConn) ServerOpt
	WithAddr   func(addr string) ServerOpt
}{
	WithSocket: func(socket net.PacketConn) ServerOpt {
		return func(s *Server) {
			s.socket = socket
		}
	},
	WithAddr: func(addr string) ServerOpt {
		return func(s *Server) {
			sock, _ := makeSocket(addr)
			s.socket = sock
		}
	},
}

// NewServer prepares a new server.
func NewServer(opts ...ServerOpt) *Server {
	ret := &Server{}
	for _, opt := range opts {
		opt(ret)
	}
	if ret.socket == nil {
		sock, _ := makeSocket("")
		ret.socket = sock
	}
	return ret
}

// Server is udp reader/writer
type Server struct {
	socket net.PacketConn
}

// Listen reads ppackets of 0x10000 max bytes.
func (s *Server) Listen(read func([]byte, *net.UDPAddr) error) error {
	// s.open.UnSet()
	var b [0x10000]byte
	for {
		n, addr, readErr := s.socket.ReadFrom(b[:])

		if readErr == nil && n == len(b) {
			readErr = fmt.Errorf("received packet exceeds buffer size")
		}

		if readErr != nil {
			if x, ok := readErr.(*net.OpError); ok && x.Temporary() == false {
				return io.EOF
			}
			log.Printf("read error: %#v\n", readErr)
			continue

		} else if read != nil {
			x := make([]byte, n) // should use buffer pool to avoid allocations, required to copy.
			copy(x, b[:n])
			go func(addr *net.UDPAddr, inflight []byte) {
				// a := *(addr.(*net.UDPAddr)) //todo: replace with an internal.
				// b := &a
				processingErr := read(inflight, addr)
				if processingErr != nil {
					log.Println("processing error:", processingErr)
				}
			}(addr.(*net.UDPAddr), x)
		}
	}
}

// Write a packet.
func (s *Server) Write(b []byte, node *net.UDPAddr) (err error) {
	n, err := s.socket.WriteTo(b, node)
	if err != nil {
		err = fmt.Errorf("error writing %d bytes to %s: %s", len(b), node, err)
	} else if n != len(b) {
		err = io.ErrShortWrite
	}
	return
}

// Close the socket.
func (s *Server) Close() (err error) {
	err = s.socket.Close()
	return
}

// Addr returns the local address.
func (s *Server) Addr() *net.UDPAddr {
	return s.socket.LocalAddr().(*net.UDPAddr)
}

func makeSocket(addr string) (socket *net.UDPConn, err error) {
	resolved, err := net.ResolveUDPAddr("", addr)
	if err != nil {
		return
	}
	socket, err = net.ListenUDP("udp", resolved)
	return
}
