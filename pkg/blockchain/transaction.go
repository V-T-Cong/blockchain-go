package blockchain

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/json"
	"math/big"
	"time"
)

type Transaction struct {
	Sender    []byte
	Receiver  []byte
	Amount    float64
	Timestamp int64
	Signature []byte
	PublicKey []byte
}

func NewTransaction(sender, receiver []byte, amount float64) *Transaction {
	return &Transaction{
		Sender:    sender,
		Receiver:  receiver,
		Amount:    amount,
		Timestamp: time.Now().Unix(),
	}
}

func (tx *Transaction) Hash() []byte {
	txCopy := *tx
	txCopy.Signature = nil
	txCopy.PublicKey = nil
	data, _ := json.Marshal(txCopy)
	hash := sha256.Sum256(data)
	return hash[:]
}

func VerifyTransaction(tx *Transaction, pubKey *ecdsa.PublicKey) bool {
	hash := tx.Hash()
	r := new(big.Int).SetBytes(tx.Signature[:len(tx.Signature)/2])
	s := new(big.Int).SetBytes(tx.Signature[len(tx.Signature)/2:])
	return ecdsa.Verify(pubKey, hash, r, s)
}
