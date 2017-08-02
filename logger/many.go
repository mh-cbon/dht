package logger

import (
	"net"

	"github.com/mh-cbon/dht/kmsg"
)

// LogReceiver is an interface to define log callbacks.
type LogReceiver interface {
	OnSendQuery(remote *net.UDPAddr, p map[string]interface{})
	OnRcvQuery(remote *net.UDPAddr, p kmsg.Msg)
	OnSendResponse(remote *net.UDPAddr, p map[string]interface{})
	OnRcvResponse(remote *net.UDPAddr, queriedQ string, queriedA map[string]interface{}, p kmsg.Msg)
	OnTxNotFound(remote *net.UDPAddr, p kmsg.Msg)
	Clear()
}

// Many loggers..
type Many struct {
	loggers []LogReceiver
}

// NewMany loggers..
func NewMany() *Many {
	return &Many{}
}

// Add LogReceiver..
func (m *Many) Add(l LogReceiver) {
	m.loggers = append(m.loggers, l)
}

// Rm LogReceiver..
func (m *Many) Rm(l LogReceiver) bool {
	var found bool
	loggers := []LogReceiver{}
	for _, n := range m.loggers {
		if n != l {
			loggers = append(loggers, n)
		} else {
			found = true
		}
	}
	return found
}

// Clear loggers..
func (m *Many) Clear() {
	for _, l := range m.loggers {
		l.Clear()
	}
}

// OnSendQuery implements LogReceiver..
func (m *Many) OnSendQuery(remote *net.UDPAddr, p map[string]interface{}) {
	for _, l := range m.loggers {
		l.OnSendQuery(remote, p)
	}
}

// OnRcvQuery implements LogReceiver..
func (m *Many) OnRcvQuery(remote *net.UDPAddr, p kmsg.Msg) {
	for _, l := range m.loggers {
		l.OnRcvQuery(remote, p)
	}
}

// OnSendResponse implements LogReceiver..
func (m *Many) OnSendResponse(remote *net.UDPAddr, p map[string]interface{}) {
	for _, l := range m.loggers {
		l.OnSendResponse(remote, p)
	}
}

// OnRcvResponse implements LogReceiver..
func (m *Many) OnRcvResponse(remote *net.UDPAddr, queriedQ string, queriedA map[string]interface{}, p kmsg.Msg) {
	for _, l := range m.loggers {
		l.OnRcvResponse(remote, queriedQ, queriedA, p)
	}
}

// OnTxNotFound implements LogReceiver..
func (m *Many) OnTxNotFound(remote *net.UDPAddr, p kmsg.Msg) {
	for _, l := range m.loggers {
		l.OnTxNotFound(remote, p)
	}
}
