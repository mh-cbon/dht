package logger

import (
	"fmt"
	"net"

	"github.com/mh-cbon/dht/kmsg"
)

// Text returns a text logger for given writer func.
func Text(writer func(string, ...interface{})) TextLogger {
	return TextLogger{
		writer: func(f string, args ...interface{}) {
			writer(f+"\n", args...)
		},
	}
}

// TextLogger is a logger helper
type TextLogger struct {
	writer func(string, ...interface{})
}

// OnSendQuery logs a query send
func (l TextLogger) OnSendQuery(remote *net.UDPAddr, p map[string]interface{}) {
	var extra string
	a := p["a"].(map[string]interface{})
	if p["q"] == kmsg.QFindNode {
		extra = fmt.Sprintf("%10v:%v", "target", ShortByteStr(a["target"]))

	} else if p["q"] == kmsg.QGetPeers {
		extra = fmt.Sprintf("%10v:%v", "info_hash", ShortByteStr(a["info_hash"]))

	} else if p["q"] == kmsg.QAnnouncePeer {
		extra = fmt.Sprintf("%10v:%v token %v port %v implied %v", "info_hash", ShortByteStr(a["info_hash"]),
			ShortStr(a["token"]), a["port"], a["implied_port"] == 0)

	} else if p["q"] == kmsg.QGet {
		extra = fmt.Sprintf("%10v:%v seq:%v", "target", ShortByteStr(a["target"]), a["seq"])

	} else if p["q"] == kmsg.QPut {
		extra = fmt.Sprintf("%10v:%q token:%v", "value", ShortStr(a["v"]), ShortByteStr(a["token"]))
		if x, ok := a["k"]; ok && len(x.([]byte)) > 0 {
			extra += fmt.Sprintf(" salt:%q seq:%2v cas:%2v k:%v sig:%v", ShortStr(a["salt"]),
				a["seq"], a["cas"], ShortByteStr(a["k"]), ShortByteStr(a["sig"]))
		}
	}
	l.writer("send quy %-24v tx:%-4x id:%q q:%-15v %v", remote.String(), p["t"], ShortByteStr(a["id"]), p["q"], extra)
}

// OnRcvQuery logs a query reception
func (l TextLogger) OnRcvQuery(remote *net.UDPAddr, p kmsg.Msg) {
	var extra string
	a := p.A
	if p.Q == kmsg.QFindNode {
		extra = fmt.Sprintf("%10v:%v", "target", ShortByteStr(a.Target))

	} else if p.Q == kmsg.QGetPeers {
		extra = fmt.Sprintf("%10v:%v", "info_hash", ShortByteStr(a.InfoHash))

	} else if p.Q == kmsg.QAnnouncePeer {
		extra = fmt.Sprintf("%10v:%v token %v port %v implied %v", "info_hash", ShortByteStr(a.InfoHash),
			a.Token, a.Port, a.ImpliedPort == 0)

	} else if p.Q == kmsg.QGet {
		extra = fmt.Sprintf("%10v:%v", "target", ShortByteStr(a.Target))

	} else if p.Q == kmsg.QPut {
		extra = fmt.Sprintf("%10v:%q token:%v", "value", ShortStr(a.V), ShortByteStr(a.Token))
		if len(a.K) > 0 {
			extra += fmt.Sprintf(" salt:%q seq:%2v cas:%2v k:%v sig:%v",
				ShortStr(a.Salt), a.Seq, a.Cas,
				ShortByteStr(a.K), ShortByteStr(a.Sign))
		}
	}
	l.writer("recv quy %-24v tx:%-4x id:%q q:%-15v %v", remote.String(), p.T, ShortByteStr(a.ID), p.Q, extra)
}

// OnSendResponse logs a response send
func (l TextLogger) OnSendResponse(remote *net.UDPAddr, p map[string]interface{}) {
	var extra string
	if p["y"] == "r" {
		r := p["r"].(kmsg.Return)
		extra = fmt.Sprintf("id:%v nodes:%3v values:%3v token:%v seq:%2v v:%q",
			ShortByteStr(r.ID), len(r.Nodes), len(r.Values), ShortByteStr(r.Token), r.Seq, ShortStr(r.V))
		if len(r.K) > 0 {
			extra += fmt.Sprintf(" k:%v sign:%v", ShortByteStr(r.K), ShortByteStr(r.Sign))
		}
	} else {
		e := p["e"].(kmsg.Error)
		extra = fmt.Sprintf("code:%v msg:%q", e.Code, e.Msg)
	}
	l.writer("send res %-24v tx:%-4x %v", remote.String(), p["t"], extra)
}

// OnRcvResponse logs a response reception
func (l TextLogger) OnRcvResponse(remote *net.UDPAddr, queriedQ string, queriedA map[string]interface{}, p kmsg.Msg) {
	var extra string
	if p.E != nil {
		e := p.E
		extra = fmt.Sprintf("code:%v msg:%q", e.Code, e.Msg)
	} else if p.Y == "r" {
		r := p.R
		extra = fmt.Sprintf("id:%v nodes:%3v values:%3v token:%v seq:%2v value:%q",
			ShortByteStr(r.ID), len(r.Nodes), len(r.Values), ShortByteStr(r.Token), r.Seq, ShortStr(r.V))
		if len(r.K) > 0 {
			extra += fmt.Sprintf(" k:%v sign:%v", ShortByteStr(r.K), ShortByteStr(r.Sign))
		}
	}
	l.writer("recv res %-24v tx:%-4x %v", remote.String(), p.T, extra)
}

// OnTxNotFound logs a response with a not found transaction (likely to happen if the node has falsely timedout)
func (l TextLogger) OnTxNotFound(remote *net.UDPAddr, p kmsg.Msg) {
}

// Clear the logger.
func (l TextLogger) Clear() {
}
