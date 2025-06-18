package wallet

import (
	"blockchain-go/pkg/blockchain"
	"blockchain-go/proto/nodepb"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
)

// func PublicKeyToAddress(pub *ecdsa.PublicKey) []byte {
// 	pubBytes := append(pub.X.Bytes(), pub.Y.Bytes()...)
// 	hash := sha256.Sum256(pubBytes)
// 	return hash[:]
// }

func SignTransaction(tx *blockchain.Transaction, privKey *ecdsa.PrivateKey) error {
	hash := tx.Hash()
	r, s, err := ecdsa.Sign(rand.Reader, privKey, hash)
	if err != nil {
		return err
	}
	tx.Signature = append(r.Bytes(), s.Bytes()...)

	tx.PublicKey = elliptic.Marshal(privKey.PublicKey.Curve, privKey.PublicKey.X, privKey.PublicKey.Y)
	// fmt.Printf("🧪 Generated PublicKey: %x\n", tx.PublicKey)
	return nil
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
