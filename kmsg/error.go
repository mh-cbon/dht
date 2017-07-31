package kmsg

import (
	"fmt"

	"github.com/anacrolix/torrent/bencode"
)

// ErrorTimeout is returned when the query times out.
var ErrorTimeout = Error{
	Code: 201,
	Msg:  "Query timeout",
}

// ErrorMethodUnknown is returned when the query verb is unknown.
var ErrorMethodUnknown = Error{
	Code: 204,
	Msg:  "Method Unknown",
}

// ErrorProtocolError is returned when a malformed incoming packet.
var ErrorProtocolError = Error{Code: 203, Msg: "Protocol Error, such as a malformed packet, invalid arguments, or bad token"}

// ErrorBadToken is returned when an unknown incoming token.
var ErrorBadToken = ErrorProtocolError

// ErrorVTooLong is returned when a V is too long (>999).
var ErrorVTooLong = Error{Code: 205, Msg: "message (v field) too big."}

// ErrorVTooShort is returned when a V is too small (<1).
var ErrorVTooShort = Error{Code: 205, Msg: "message (v field) too small."}

// ErrorInvalidSig is returned when a signature is invalid.
var ErrorInvalidSig = Error{Code: 206, Msg: "invalid signature"}

// ErrorNoK is returned when k is missing.
var ErrorNoK = Error{Code: 206, Msg: "invalid k"}

// ErrorSaltTooLong is returned when a salt is too long (>64).
var ErrorSaltTooLong = Error{Code: 207, Msg: "salt (salt field) too big."}

// ErrorCasMismatch is returned when the cas mismatch (put).
var ErrorCasMismatch = Error{Code: 301, Msg: "the CAS hash mismatched, re-read value and try again."}

// ErrorSeqLessThanCurrent is returned when the incoming seq is less than current (get/put).
var ErrorSeqLessThanCurrent = Error{Code: 302, Msg: "sequence number less than current."}

// ErrorInternalIssue is returned when an internal error occurs.
var ErrorInternalIssue = Error{Code: 501, Msg: "an internal error prevented the operation to succeed."}

// ErrorInsecureNodeID is returned when an incoming query with an insecure id is detected.
var ErrorInsecureNodeID = Error{Code: 305, Msg: "Invalid node id."}

// Error Represented as a string or list in bencode.
type Error struct {
	Code int
	Msg  string
}

var (
	_ bencode.Unmarshaler = (*Error)(nil)
	_ bencode.Marshaler   = (*Error)(nil)
	_ error               = Error{}
)

// UnmarshalBencode implements bUnmarshalling.
func (e *Error) UnmarshalBencode(_b []byte) (err error) {
	var _v interface{}
	err = bencode.Unmarshal(_b, &_v)
	if err != nil {
		return
	}
	switch v := _v.(type) {
	case []interface{}:
		func() {
			defer func() {
				r := recover()
				if r == nil {
					return
				}
				err = fmt.Errorf("unpacking %#v: %s", v, r)
			}()
			e.Code = int(v[0].(int64))
			e.Msg = v[1].(string)
		}()
	case string:
		e.Msg = v
	default:
		err = fmt.Errorf(`KRPC error bencode value has unexpected type: %T`, _v)
	}
	return
}

// MarshalBencode implements bMarshalling.
func (e Error) MarshalBencode() (ret []byte, err error) {
	return bencode.Marshal([]interface{}{e.Code, e.Msg})
}

func (e Error) Error() string {
	return fmt.Sprintf("KRPC error %d: %s", e.Code, e.Msg)
}
