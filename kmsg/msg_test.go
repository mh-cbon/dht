package kmsg

import (
	"bytes"
	"net"
	"testing"

	"github.com/anacrolix/torrent/bencode"
	"github.com/mh-cbon/dht/util"
)

func testMarshalUnmarshalMsg(t *testing.T, m Msg, expected string) {
	b, err := bencode.Marshal(m)
	if err != nil {
		t.Errorf("Wanted err=nil, got=%v", err)
	}
	if expected != string(b) {
		t.Errorf("Wanted expected=%x, got=%x", expected, string(b))
	}

	var _m Msg
	err = bencode.Unmarshal([]byte(expected), &_m)
	if err != nil {
		t.Errorf("Wanted err=nil, got=%v", err)
	}
	if m.Y != _m.Y {
		t.Errorf("Wanted m.Y=%v, got=%v", m.Y, _m.Y)
	}
	if m.Q != _m.Q {
		t.Errorf("Wanted m.Q=%v, got=%v", m.Q, _m.Q)
	}
	if m.T != _m.T {
		t.Errorf("Wanted m.T=%v, got=%v", m.T, _m.T)
	}

	if m.A == nil && _m.A != nil {
		t.Errorf("Wanted m.A=nil, got=%v", _m.A)
	} else if m.A != nil && _m.A == nil {
		t.Errorf("Wanted m.A!=nil, got=%v", _m.A)
	} else if m.A != nil && _m.A != nil {
		if m.A.Cas != _m.A.Cas {
			t.Errorf("Wanted m.A.Cas=%v, got=%v", m.A.Cas, _m.A.Cas)
		} else if m.A.Seq != _m.A.Seq {
			t.Errorf("Wanted m.A.Seq=%v, got=%v", m.A.Seq, _m.A.Seq)
		} else if m.A.ID != _m.A.ID {
			t.Errorf("Wanted m.A.ID=%v, got=%v", m.A.ID, _m.A.ID)
		} else if m.A.ImpliedPort != _m.A.ImpliedPort {
			t.Errorf("Wanted m.A.ImpliedPort=%v, got=%v", m.A.ImpliedPort, _m.A.ImpliedPort)
		} else if m.A.InfoHash != _m.A.InfoHash {
			t.Errorf("Wanted m.A.InfoHash=%v, got=%v", m.A.InfoHash, _m.A.InfoHash)
		} else if m.A.Port != _m.A.Port {
			t.Errorf("Wanted m.A.Port=%v, got=%v", m.A.Port, _m.A.Port)
		} else if m.A.Salt != _m.A.Salt {
			t.Errorf("Wanted m.A.Salt=%v, got=%v", m.A.Salt, _m.A.Salt)
		} else if m.A.Target != _m.A.Target {
			t.Errorf("Wanted m.A.Target=%v, got=%v", m.A.Target, _m.A.Target)
		} else if m.A.Token != _m.A.Token {
			t.Errorf("Wanted m.A.Token=%v, got=%v", m.A.Token, _m.A.Token)
		} else if m.A.V != _m.A.V {
			t.Errorf("Wanted m.A.V=%v, got=%v", m.A.V, _m.A.V)
		} else if !bytes.Equal(m.A.K, _m.A.K) {
			t.Errorf("Wanted m.A.k=%v, got=%v", m.A.K, _m.A.K)
		} else if !bytes.Equal(m.A.Sign, _m.A.Sign) {
			t.Errorf("Wanted m.A.Sign=%v, got=%v", m.A.Sign, _m.A.Sign)
		}
	}

	if m.R == nil && _m.R != nil {
		t.Errorf("Wanted m.R=nil, got=%v", _m.R)
	} else if m.R != nil && _m.R == nil {
		t.Errorf("Wanted m.R!=nil, got=%v", _m.R)
	} else if m.R != nil && _m.R != nil {
		if m.R.ID != _m.R.ID {
			t.Errorf("Wanted m.R.ID=%v, got=%v", m.R.ID, _m.R.ID)
		} else if m.R.Seq != _m.R.Seq {
			t.Errorf("Wanted m.R.Seq=%v, got=%v", m.R.Seq, _m.R.Seq)
		} else if m.R.Token != _m.R.Token {
			t.Errorf("Wanted m.R.Token=%v, got=%v", m.R.Token, _m.R.Token)
		} else if m.R.V != _m.R.V {
			t.Errorf("Wanted m.R.V=%v, got=%v", m.R.V, _m.R.V)
		} else if len(m.R.Nodes) != len(_m.R.Nodes) {
			t.Errorf("Wanted len(m.R.Nodes)=%v, got=%v", len(m.R.Nodes), len(_m.R.Nodes))
		} else if len(m.R.Values) != len(_m.R.Values) {
			t.Errorf("Wanted len(m.R.Values)=%v, got=%v", len(m.R.Values), len(_m.R.Values))
		} else {
			for i := range m.R.Nodes {
				left := m.R.Nodes[i]
				right := _m.R.Nodes[i]
				if left.Addr.String() != right.Addr.String() {
					t.Errorf("Wanted m.R.Nodes[i].Addr=%v, got=%v", left.Addr.String(), right.Addr.String())
				} else if !bytes.Equal(left.ID[:], right.ID[:]) {
					t.Errorf("Wanted m.R.Nodes[i].ID=%v, got=%v", left.ID, right.ID)
				}
			}
			for i := range m.R.Values {
				left := m.R.Values[i]
				right := _m.R.Values[i]
				if left.Port != right.Port {
					t.Errorf("Wanted m.R.Values[i].Port=%v, got=%v", left.Port, right.Port)
				} else if left.IP.String() != right.IP.String() {
					t.Errorf("Wanted m.R.Values[i].IP=%v, got=%v", left.IP.String(), right.IP.String())
				}
			}
		}
	}

	if m.E == nil && _m.E != nil {
		t.Errorf("Wanted m.E=nil, got=%v", _m.E)
	} else if m.E != nil && _m.E == nil {
		t.Errorf("Wanted m.E!=nil, got=%v", _m.E)
	} else if m.E != nil && _m.E != nil {
		if m.E.Code != _m.E.Code {
			t.Errorf("Wanted m.E.Code=%v, got=%v", m.E.Code, _m.E.Code)
		} else if m.E.Msg != _m.E.Msg {
			t.Errorf("Wanted m.E.Msg=%v, got=%v", m.E.Msg, _m.E.Msg)
		}
	}

	if m.IP.IP.String() != _m.IP.IP.String() {
		t.Errorf("Wanted m.IP=%v, got=%v", m.IP, _m.IP)
	}
}

func TestMarshalUnmarshalMsg(t *testing.T) {
	testMarshalUnmarshalMsg(t, Msg{}, "d1:t0:1:y0:e")
	testMarshalUnmarshalMsg(t, Msg{
		Y: "q",
		Q: "ping",
		T: "hi",
	}, "d1:q4:ping1:t2:hi1:y1:qe")
	testMarshalUnmarshalMsg(t, Msg{
		Y: "e",
		T: "42",
		E: &Error{Code: 200, Msg: "nop"},
	}, "d1:eli200e3:nope1:t2:421:y1:ee")
	testMarshalUnmarshalMsg(t, Msg{
		Y: "r",
		T: "\x8c%",
		R: &Return{},
	}, "d1:rd2:id0:e1:t2:\x8c%1:y1:re")
	testMarshalUnmarshalMsg(t, Msg{
		Y: "r",
		T: "\x8c%",
		R: &Return{
			Nodes: CompactIPv4NodeInfo{
				NodeInfo{
					Addr: &net.UDPAddr{
						IP:   net.IPv4(1, 2, 3, 4).To4(),
						Port: 0x1234,
					},
				},
			},
		},
	}, "d1:rd2:id0:5:nodes26:\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x02\x03\x04\x124e1:t2:\x8c%1:y1:re")
	testMarshalUnmarshalMsg(t, Msg{
		Y: "r",
		T: "\x8c%",
		R: &Return{
			Values: []util.CompactPeer{
				{
					IP:   net.IPv4(1, 2, 3, 4).To4(),
					Port: 0x5678,
				},
			},
		},
	}, "d1:rd2:id0:6:valuesl6:\x01\x02\x03\x04\x56\x78ee1:t2:\x8c%1:y1:re")
	testMarshalUnmarshalMsg(t, Msg{
		Y: "r",
		T: "\x03",
		R: &Return{
			ID: "\xeb\xff6isQ\xffJ\xec)ͺ\xab\xf2\xfb\xe3F|\xc2g",
		},
		IP: util.CompactPeer{
			IP:   net.IPv4(124, 168, 180, 8).To4(),
			Port: 62844,
		},
	}, "d2:ip6:|\xa8\xb4\b\xf5|1:rd2:id20:\xeb\xff6isQ\xffJ\xec)ͺ\xab\xf2\xfb\xe3F|\xc2ge1:t1:\x031:y1:re")
}

func TestUnmarshalGetPeersResponse(t *testing.T) {
	var msg Msg
	err := bencode.Unmarshal([]byte("d1:rd6:valuesl6:\x01\x02\x03\x04\x05\x066:\x07\x08\x09\x0a\x0b\x0ce5:nodes52:\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x02\x03\x04\x05\x06\x07\x08\x09\x02\x03\x04\x05\x06\x07\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x02\x03\x04\x05\x06\x07\x08\x09\x02\x03\x04\x05\x06\x07ee"), &msg)
	if err != nil {
		t.Errorf("Wanted err=nil, got=%v", err)
	}
	if len(msg.R.Values) != 2 {
		t.Errorf("Wanted len(msg.R.Values)=2, got=%v", len(msg.R.Values))
	}
	if len(msg.R.Nodes) != 2 {
		t.Errorf("Wanted len(msg.R.Nodes)=2, got=%v", len(msg.R.Nodes))
	}
	if msg.E != nil {
		t.Errorf("Wanted msg.E=nil, got=%v", msg.E)
	}
}
