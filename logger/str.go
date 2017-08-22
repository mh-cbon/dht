package logger

import (
	"fmt"
	"strings"
)

// ShortByteStr turns a byte string into hex, then shortens it.
func ShortByteStr(f interface{}) string {
	var s string
	if f == nil {
		return ""
	} else if x, ok := f.(string); ok {
		s = fmt.Sprintf("%x", string(x))
	} else if x, ok := f.([]byte); ok {
		s = fmt.Sprintf("%x", string(x))
	}
	return ShortStr(s)
}

// ShortStr shorten a string to display: head...tail.
func ShortStr(f interface{}) string {
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
