package stats

import (
	"net"
	"time"

	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/security"
)

// Activity is a timestamp for one of query, response, error.
type Activity struct {
	kind string
	id   string // receive query
	ro   bool   // receive query
	err  kmsg.Error
	date time.Time
}

// Is one of given kinds.
func (p Activity) Is(kinds ...string) bool {
	for _, what := range kinds {
		if what == p.kind {
			return true
		}
	}
	return false
}

// IsErr and one of given codes.
func (p Activity) IsErr(code ...int) bool {
	for _, c := range code {
		if p.kind == "e" && p.err.Code == c {
			return true
		}
	}
	return false
}

// IsSecured id for given ip.
func (p Activity) IsSecured(ip net.IP) bool {
	return security.NodeIDSecure(p.id, ip)
}

// IsActive if it occurred within given duration
func (p Activity) IsActive(d time.Duration) bool {
	return p.date.After(time.Now().Add(d))
}
