package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
)

type PrivateKey = ecdsa.PrivateKey

func GenerateKeyPair() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}
