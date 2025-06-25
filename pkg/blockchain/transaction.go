package blockchain

import (
	"blockchain-go/pkg/mpt"

	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
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

// Chuyển tiền và cập nhật MPT trạng thái
func ApplyTransaction(state *mpt.MPT, tx *Transaction) error {
	// Lấy số dư người gửi
	senderBalBytes, _ := state.Get([]byte(tx.Sender))
	senderBal := BytesToUint64(senderBalBytes)

	// Kiểm tra đủ tiền
	if senderBal < uint64(tx.Amount) {
		return errors.New("insufficient balance")
	}

	// Lấy số dư người nhận
	receiverBalBytes, _ := state.Get([]byte(tx.Receiver))
	receiverBal := BytesToUint64(receiverBalBytes)

	// Tính toán lại số dư
	newSenderBal := senderBal - uint64(tx.Amount)
	newReceiverBal := receiverBal + uint64(tx.Amount)

	// Cập nhật lại vào MPT
	state.Insert([]byte(tx.Sender), Uint64ToBytes(newSenderBal))
	state.Insert([]byte(tx.Receiver), Uint64ToBytes(newReceiverBal))

	return nil
}

func Uint64ToBytes(n uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, n)
	return buf
}

func BytesToUint64(b []byte) uint64 {
	if len(b) < 8 {
		return 0
	}
	return binary.BigEndian.Uint64(b)
}
