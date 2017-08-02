package stats

import (
	"net"
	"time"

	"github.com/mh-cbon/dht/kmsg"
)

// NewPeer for givne addr.
func NewPeer(remote net.UDPAddr) *Peer {
	return &Peer{remote, []*Activity{}}
}

//Peer is a remote addr with activities.
type Peer struct {
	addr     net.UDPAddr
	activity []*Activity
}

// Cap activities to given max length
func (p *Peer) Cap(max int) {
	if len(p.activity) > max {
		p.activity = p.activity[0:max]
	}
}

// OnRcvQuery updates activity log.
func (p *Peer) OnRcvQuery(remote *net.UDPAddr, query kmsg.Msg) {
	a := &Activity{kind: "q", id: query.A.ID, ro: query.RO == 1, date: time.Now()}
	p.activity = append([]*Activity{a}, p.activity...)
}

// OnSendResponse updates activity log.
func (p *Peer) OnSendResponse(remote *net.UDPAddr, q map[string]interface{}) {}

// OnSendError updates activity log.
func (p *Peer) OnSendError(remote *net.UDPAddr, q map[string]interface{}) {}

// OnRcvResponse updates activity log.
func (p *Peer) OnRcvResponse(remote *net.UDPAddr, queriedQ string, queriedA map[string]interface{}, response kmsg.Msg) {
	var a *Activity
	if response.E != nil {
		a = &Activity{kind: "e", err: *response.E, date: time.Now()}
	} else {
		a = &Activity{kind: "r", id: response.R.ID, date: time.Now()}
	}
	p.activity = append([]*Activity{a}, p.activity...)
}

// OnTxNotFound updates activity log.
func (p *Peer) OnTxNotFound(remote *net.UDPAddr, q kmsg.Msg) {
}

// IsTimeout if last 3 activity are timeout errors.
func (p *Peer) IsTimeout() bool {
	i := 0
	for _, a := range p.activity {
		if a.IsErr(kmsg.ErrorTimeout.Code) {
			i++
		} else {
			break
		}
	}
	return i > 2
}

// IsActive if it has a query or a response within given d duration
func (p *Peer) IsActive(d time.Duration) bool {
	for _, a := range p.activity {
		if a.Is("q", "r") && a.IsActive(d) {
			return true //todo: check the change.
		}
	}
	return false
}

// IsRO if it contains a query set to ro.
func (p *Peer) IsRO() bool {
	for _, a := range p.activity {
		if a.Is("q") {
			return a.ro
		}
	}
	return false
}

// LastIDValid from last query / last response.
func (p *Peer) LastIDValid() bool {
	for _, a := range p.activity {
		if a.Is("q", "r") {
			return a.IsSecured(p.addr.IP)
		}
	}
	return true
}
