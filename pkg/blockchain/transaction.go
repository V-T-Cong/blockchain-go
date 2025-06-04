package blockchain

import (
	"crypto/sha256"
	"encoding/json"
	"time"
)

type Transaction struct {
	Sender    []byte
	Receiver  []byte
	Amount    float64
	Timestamp int64
	Signature []byte
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
	data, _ := json.Marshal(txCopy)
	hash := sha256.Sum256(data)
	return hash[:]
}
