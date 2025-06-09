package cryptohelper

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
)

func BytesToPublicKey(pubKeyBytes []byte) (*ecdsa.PublicKey, error) {
	x, y := elliptic.Unmarshal(elliptic.P256(), pubKeyBytes)
	if x == nil || y == nil {
		return nil, errors.New("invalid public key bytes")
	}

	pubKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}

	return pubKey, nil
}
