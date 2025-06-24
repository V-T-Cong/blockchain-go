package validation

import (
	"blockchain-go/pkg/blockchain"
	"blockchain-go/pkg/cryptohelper"
	"blockchain-go/pkg/mpt"
	"blockchain-go/pkg/state"
	"bytes"
	"encoding/hex"
	"fmt"
)

func ValidateBlock(block *blockchain.Block, stateManager *state.State, latestBlock *blockchain.Block) error {
	// 1. Kiểm tra số dư và chữ ký của từng giao dịch
	for _, tx := range block.Transactions {
		// Bỏ qua giao dịch genesis
		if string(tx.Sender) == "GENESIS" {
			continue
		}

		// Kiểm tra chữ ký
		pubKey, err := cryptohelper.BytesToPublicKey(tx.PublicKey)
		if err != nil {
			return fmt.Errorf("lỗi chuyển đổi public key: %w", err)
		}
		if !blockchain.VerifyTransaction(tx, pubKey) {
			return fmt.Errorf("chữ ký không hợp lệ trong giao dịch")
		}

		// Kiểm tra số dư
		senderKey := hex.EncodeToString(tx.Sender)
		balance, err := stateManager.GetBalance(senderKey)
		if err != nil {
			return fmt.Errorf("không thể lấy số dư của người gửi %s: %w", senderKey, err)
		}
		if balance < tx.Amount {
			return fmt.Errorf("người gửi %s không đủ số dư (có %f, cần %f)", senderKey, balance, tx.Amount)
		}
	}

	// 2. Kiểm tra Merkle Root
	var txHashes [][]byte
	for _, tx := range block.Transactions {
		txHashes = append(txHashes, tx.Hash())
	}
	_, computedRoot := mpt.BuildMPTFromTxHashes(txHashes)
	if !bytes.Equal(computedRoot, block.MerkleRoot) {
		return fmt.Errorf("merkle root không khớp (kỳ vọng %x, nhận được %x)", block.MerkleRoot, computedRoot)
	}

	// 3. Kiểm tra Previous Block Hash (bỏ qua cho block genesis)
	if latestBlock != nil {
		if !bytes.Equal(block.PreviousBlockHash, latestBlock.CurrentBlockHash) {
			return fmt.Errorf("previous block hash không khớp")
		}
	}

	return nil
}
