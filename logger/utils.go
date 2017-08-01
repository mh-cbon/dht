package logger

import (
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/mh-cbon/dht/kmsg"
	"github.com/mh-cbon/dht/rpc"
)

// LogFuncs is a logger helper
type LogFuncs struct{}

// OnSendQuery logs a query send
func (l LogFuncs) OnSendQuery(remote *net.UDPAddr, p map[string]interface{}) {
	var extra string
	a := p["a"].(map[string]interface{})
	if p["q"] == rpc.QFindNode {
		extra = fmt.Sprintf("%10v:%v", "target", shortByteStr(a["target"]))

	} else if p["q"] == rpc.QGetPeers {
		extra = fmt.Sprintf("%10v:%v", "info_hash", shortByteStr(a["info_hash"]))

	} else if p["q"] == rpc.QAnnouncePeer {
		extra = fmt.Sprintf("%10v:%v token %v port %v implied %v", "info_hash", shortByteStr(a["info_hash"]),
			shortStr(a["token"]), a["port"], a["implied_port"] == 0)

	} else if p["q"] == rpc.QGet {
		extra = fmt.Sprintf("%10v:%v", "target", shortByteStr(a["target"]))

	} else if p["q"] == rpc.QPut {
		extra = fmt.Sprintf("%10v:%q token:%v", "value", shortStr(a["v"]), shortByteStr(a["token"]))
		if x, ok := a["k"]; ok && len(x.([]byte)) > 0 {
			extra += fmt.Sprintf(" salt:%q seq:%2v cas:%2v k:%v sig:%v", shortStr(a["salt"]),
				a["seq"], a["cas"], shortByteStr(a["k"]), shortByteStr(a["sig"]))
		}
	}
	log.Printf("send quy %-24v tx:%-4x id:%q q:%-15v %v", remote.String(), p["t"], shortByteStr(a["id"]), p["q"], extra)
}

// OnRcvQuery logs a query reception
func (l LogFuncs) OnRcvQuery(remote *net.UDPAddr, p kmsg.Msg) {
	var extra string
	a := p.A
	if p.Q == rpc.QFindNode {
		extra = fmt.Sprintf("%10v:%v", "target", shortByteStr(a.Target))

	} else if p.Q == rpc.QGetPeers {
		extra = fmt.Sprintf("%10v:%v", "info_hash", shortByteStr(a.InfoHash))

	} else if p.Q == rpc.QAnnouncePeer {
		extra = fmt.Sprintf("%10v:%v token %v port %v implied %v", "info_hash", shortByteStr(a.InfoHash),
			a.Token, a.Port, a.ImpliedPort == 0)

	} else if p.Q == rpc.QGet {
		extra = fmt.Sprintf("%10v:%v", "target", shortByteStr(a.Target))

	} else if p.Q == rpc.QPut {
		extra = fmt.Sprintf("%10v:%q token:%v", "value", shortStr(a.V), shortByteStr(a.Token))
		if len(a.K) > 0 {
			extra += fmt.Sprintf(" salt:%q seq:%2v cas:%2v k:%v sig:%v",
				shortStr(a.Salt), a.Seq, a.Cas,
				shortByteStr(a.K), shortByteStr(a.Sign))
		}
	}
	log.Printf("recv quy %-24v tx:%-4x id:%q q:%-15v %v", remote.String(), p.T, shortByteStr(a.ID), p.Q, extra)
}

// OnSendResponse logs a response send
func (l LogFuncs) OnSendResponse(remote *net.UDPAddr, p map[string]interface{}) {
	var extra string
	if p["y"] == "r" {
		r := p["r"].(kmsg.Return)
		extra = fmt.Sprintf("id:%v nodes:%3v values:%3v token:%v seq:%2v value:%q",
			shortByteStr(r.ID), len(r.Nodes), len(r.Values), shortByteStr(r.Token), r.Seq, shortStr(r.V))
		if len(r.K) > 0 {
			extra += fmt.Sprintf(" k:%v sign:%v", shortByteStr(r.K), shortByteStr(r.Sign))
		}
	} else {
		e := p["e"].(kmsg.Error)
		extra = fmt.Sprintf("code:%v msg:%q", e.Code, e.Msg)
	}
	log.Printf("send res %-24v tx:%-4x %v", remote.String(), p["t"], extra)
}

// OnRcvResponse logs a response reception
func (l LogFuncs) OnRcvResponse(remote *net.UDPAddr, p kmsg.Msg) {
	var extra string
	if p.E != nil {
		e := p.E
		extra = fmt.Sprintf("code:%v msg:%q", e.Code, e.Msg)
	} else if p.Y == "r" {
		r := p.R
		extra = fmt.Sprintf("id:%v nodes:%3v values:%3v token:%v seq:%2v value:%q",
			shortByteStr(r.ID), len(r.Nodes), len(r.Values), shortByteStr(r.Token), r.Seq, shortStr(r.V))
		if len(r.K) > 0 {
			extra += fmt.Sprintf(" k:%v sign:%v", shortByteStr(r.K), shortByteStr(r.Sign))
		}
	}
	log.Printf("recv res %-24v tx:%-4x %v", remote.String(), p.T, extra)
}

// Std helper for logging
var Std = LogFuncs{}

func shortByteStr(f interface{}) string {
	var s string
	if f == nil {
		return ""
	} else if x, ok := f.(string); ok {
		s = fmt.Sprintf("%x", string(x))
	} else if x, ok := f.([]byte); ok {
		s = fmt.Sprintf("%x", string(x))
	}
	return shortStr(s)
}

func shortStr(f interface{}) string {
	var s string
	if f == nil {
		return ""
	} else if x, ok := f.(string); ok {
		s = x
	} else if x, ok := f.([]byte); ok {
		s = string(x)
	}
	if len(s) > 8 {
		d := len(s) - 8
		if d > 3 {
			d = 3
		}
		return s[:4] + strings.Repeat(".", d) + s[len(s)-4:]
	}
	return s
}
