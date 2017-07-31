package crypto

import (
	"crypto/sha1"
	"fmt"
)

// HashSha1 hashes many string to a string.
func HashSha1(s ...string) string {
	h := sha1.New()
	for _, v := range s {
		h.Write([]byte(v))
	}
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs)
}
