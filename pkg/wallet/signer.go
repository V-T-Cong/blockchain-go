package wallet

import (
	"blockchain-go/pkg/blockchain"
	"blockchain-go/proto/nodepb"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
)

func PublicKeyToAddress(pub *ecdsa.PublicKey) []byte {
	pubBytes := append(pub.X.Bytes(), pub.Y.Bytes()...)
	hash := sha256.Sum256(pubBytes)
	return hash[:]
}

func SignTransaction(tx *blockchain.Transaction, privKey *ecdsa.PrivateKey) error {
	hash := tx.Hash()
	r, s, err := ecdsa.Sign(rand.Reader, privKey, hash)
	if err != nil {
		return err
	}
	tx.Signature = append(r.Bytes(), s.Bytes()...)

	tx.PublicKey = elliptic.Marshal(privKey.PublicKey.Curve, privKey.PublicKey.X, privKey.PublicKey.Y)
	return nil
}

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

func VerifyTransaction(tx *blockchain.Transaction, pubKey *ecdsa.PublicKey) bool {
	hash := tx.Hash()
	r := new(big.Int).SetBytes(tx.Signature[:len(tx.Signature)/2])
	s := new(big.Int).SetBytes(tx.Signature[len(tx.Signature)/2:])
	return ecdsa.Verify(pubKey, hash, r, s)
}

// Hash only the important transaction fields
func HashTransactionFields(tx *nodepb.Transaction) []byte {
	data := append(tx.Sender, tx.Receiver...)
	data = append(data, []byte(fmt.Sprintf("%.2f%d", tx.Amount, tx.Timestamp))...)
	hash := sha256.Sum256(data)
	return hash[:]
}

// Separate function for signing a hash
func SignHash(hash []byte, privKey *ecdsa.PrivateKey) (*big.Int, *big.Int, error) {
	return ecdsa.Sign(rand.Reader, privKey, hash)
}
