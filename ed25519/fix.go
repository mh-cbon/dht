package ed25519

import (
	"crypto/sha512"

	edwards25519 "github.com/mh-cbon/dht/ed25519/internaledwards25519"
	src "golang.org/x/crypto/ed25519"
)

// see https://stackoverflow.com/questions/44810708/ed25519-public-result-is-different

// PrivateKey is an alias
type PrivateKey src.PrivateKey

// GenerateKey is an alias
var GenerateKey = src.GenerateKey

// Verify is an alias
var Verify = src.Verify

// PublicKeyFromPvk generates the [32]byte public key
// corresponding to the already hashed private key.
//
// This code is mostly copied from GenerateKey in the
// golang.org/x/crypto/ed25519 package, from after the SHA512
// calculation of the seed.
func PublicKeyFromPvk(privateKey []byte) []byte {
	var A edwards25519.ExtendedGroupElement
	var hBytes [32]byte
	copy(hBytes[:], privateKey)
	edwards25519.GeScalarMultBase(&A, &hBytes)
	var publicKeyBytes [32]byte
	A.ToBytes(&publicKeyBytes)

	return publicKeyBytes[:]
}

// Sign calculates the signature from the (pre hashed) private key, public key and message.
//
// This code is mostly copied from the Sign function from
// golang.org/x/crypto/ed25519, from after the SHA512 calculation of the
// seed.
func Sign(privateKey, publicKey, message []byte) []byte {

	var privateKeyA [32]byte
	copy(privateKeyA[:], privateKey) // we need this in an array later
	var messageDigest, hramDigest [64]byte

	h := sha512.New()
	h.Write(privateKey[32:])
	h.Write(message)
	h.Sum(messageDigest[:0])

	var messageDigestReduced [32]byte
	edwards25519.ScReduce(&messageDigestReduced, &messageDigest)
	var R edwards25519.ExtendedGroupElement
	edwards25519.GeScalarMultBase(&R, &messageDigestReduced)

	var encodedR [32]byte
	R.ToBytes(&encodedR)

	h.Reset()
	h.Write(encodedR[:])
	h.Write(publicKey)
	h.Write(message)
	h.Sum(hramDigest[:0])
	var hramDigestReduced [32]byte
	edwards25519.ScReduce(&hramDigestReduced, &hramDigest)

	var s [32]byte
	edwards25519.ScMulAdd(&s, &hramDigestReduced, &privateKeyA, &messageDigestReduced)

	signature := make([]byte, 64)
	copy(signature[:], encodedR[:])
	copy(signature[32:], s[:])

	return signature
}
