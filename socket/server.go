package socket

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/tevino/abool"
)

// Server is udp reader/writer
type Server struct {
	socket net.PacketConn
	config ServerConfig
	open   *abool.AtomicBool
}

// NewServer prepares a new server.
func NewServer(config ServerConfig) *Server {
	ret := &Server{
		config: config,
		open:   abool.New(),
	}
	sock, _ := makeSocket(config.Addr)
	ret.socket = sock
	return ret
}

// Listen reads ppackets of 0x10000 max bytes.
func (s *Server) Listen(read func([]byte, *net.UDPAddr) error) error {
	s.open.UnSet()
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

// ServerConfig allows to set up a  configuration of the `Server` instance
// to be created with NewServer
type ServerConfig struct {
	Addr string
}

// WithAddr of the socket.
func (c ServerConfig) WithAddr(addr string) ServerConfig {
	c.Addr = addr
	return c
}

func makeSocket(addr string) (socket *net.UDPConn, err error) {
	resolved, err := net.ResolveUDPAddr("", addr)
	if err != nil {
		return
	}
	socket, err = net.ListenUDP("udp", resolved)
	return
}
