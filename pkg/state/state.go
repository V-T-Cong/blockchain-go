package state

import (
	"blockchain-go/pkg/blockchain"
	"blockchain-go/pkg/storage"
	"encoding/hex"
	"errors"
	"fmt"
	// "log"
	"strconv"

	"github.com/syndtr/goleveldb/leveldb" // SỬA ĐỔI: Import package chính, không phải 'opt'
)

// State quản lý số dư của các tài khoản
type State struct {
	db *storage.DB
}

// NewState tạo một State Manager mới
func NewState(db *storage.DB) (*State, error) {
	s := &State{db: db}
	return s, nil
}

// GetBalance lấy số dư của một địa chỉ (dạng chuỗi hex)
func (s *State) GetBalance(address string) (float64, error) {
	key := []byte("balance-" + address)
	data, err := s.db.Get(key)
	if err != nil {
		// SỬA ĐỔI: Sử dụng leveldb.ErrNotFound
		if errors.Is(err, leveldb.ErrNotFound) {
			return 0, nil // Nếu không tìm thấy, số dư là 0
		}
		return 0, err // Lỗi khác
	}

	balance, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse balance: %w", err)
	}
	return balance, nil
}

// SetBalance đặt số dư cho một địa chỉ (dạng chuỗi hex)
func (s *State) SetBalance(address string, balance float64) error {
	key := []byte("balance-" + address)
	value := []byte(strconv.FormatFloat(balance, 'f', -1, 64))
	return s.db.Put(key, value, nil)
}

// ApplyTransaction cập nhật số dư dựa trên một giao dịch.
func (s *State) ApplyTransaction(tx *blockchain.Transaction) error {
	// Xử lý trường hợp người gửi là giao dịch GENESIS
	if string(tx.Sender) == "GENESIS" {
		receiverKey := hex.EncodeToString(tx.Receiver)
		// log.Printf("--- DEBUG [ApplyTx]: Applying GENESIS tx. Receiver: %s, Amount: %f", receiverKey, tx.Amount) // DEBUG

		receiverBalance, err := s.GetBalance(receiverKey)
		if err != nil {
			return fmt.Errorf("failed to get receiver balance for genesis tx: %w", err)
		}
		// log.Printf("--- DEBUG [ApplyTx]: Old balance: %f", receiverBalance) // DEBUG

		newBalance := receiverBalance + tx.Amount
		if err := s.SetBalance(receiverKey, newBalance); err != nil {
			return fmt.Errorf("failed to set receiver balance for genesis tx: %w", err)
		}
		// log.Printf("--- DEBUG [ApplyTx]: New balance set to: %f", newBalance) // DEBUG
		return nil
	}

	// Xử lý giao dịch thông thường
	senderKey := hex.EncodeToString(tx.Sender)
	receiverKey := hex.EncodeToString(tx.Receiver)

	senderBalance, err := s.GetBalance(senderKey)
	if err != nil {
		return fmt.Errorf("failed to get sender balance: %w", err)
	}

	if senderBalance < tx.Amount {
		return fmt.Errorf("insufficient funds for sender %s", senderKey)
	}

	receiverBalance, err := s.GetBalance(receiverKey)
	if err != nil {
		return fmt.Errorf("failed to get receiver balance: %w", err)
	}

	// Cập nhật và lưu lại
	if err := s.SetBalance(senderKey, senderBalance-tx.Amount); err != nil {
		return err
	}
	if err := s.SetBalance(receiverKey, receiverBalance+tx.Amount); err != nil {
		return err
	}
	return nil
}

// RebuildStateFromBlockchain quét toàn bộ blockchain để tính toán lại trạng thái số dư.
func (s *State) RebuildStateFromBlockchain() error {
	fmt.Println("Rebuilding state from blockchain...")

	latestBlock, err := s.db.GetLatestBlock()
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			fmt.Println("No blocks in DB, state is empty.")
			return nil
		}
		return fmt.Errorf("failed to get latest block for state rebuild: %w", err)
	}

	for i := 0; i <= int(latestBlock.Height); i++ {
		block, err := s.db.GetBlockByHeight(i)
		if err != nil {
			return fmt.Errorf("failed to get block %d for state rebuild: %w", i, err)
		}

		for _, tx := range block.Transactions {

			// // logs Debug thông tin giao dịch
			// log.Printf("--- DEBUG [Rebuild]: Processing Tx in Block %d ---", block.Height)
			// log.Printf("--- DEBUG [Rebuild]: Sender: %s", string(tx.Sender))
			// log.Printf("--- DEBUG [Rebuild]: Receiver (hex): %s", hex.EncodeToString(tx.Receiver))
			// log.Printf("--- DEBUG [Rebuild]: Amount: %f", tx.Amount)

			if err := s.ApplyTransaction(tx); err != nil {
				fmt.Printf("❌ Failed to apply tx during rebuild (block %d): %v\n", block.Height, err)
			}
		}
	}
	fmt.Println("✅ State rebuild complete.")
	return nil
}
