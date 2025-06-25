package blockchain

import (
	"blockchain-go/pkg/cryptohelper"
	"blockchain-go/pkg/mpt"
	"blockchain-go/proto/nodepb"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

type Block struct {
	Height            int64
	Transactions      []*Transaction
	MerkleRoot        []byte
	PreviousBlockHash []byte
	CurrentBlockHash  []byte
	Timestamp         int64
}

func NewBlock(transactions []*Transaction, previousBlockHash []byte, height int) *Block {
	var txHashes [][]byte
	for _, tx := range transactions {
		txHashes = append(txHashes, tx.Hash())
	}
	_, mptRoot := mpt.BuildMPTFromTxHashes(txHashes)

	block := &Block{
		Height:            int64(height),
		Transactions:      transactions,
		MerkleRoot:        mptRoot,
		PreviousBlockHash: previousBlockHash,
		Timestamp:         time.Now().Unix(),
	}

	block.CurrentBlockHash = block.Hash()
	return block
}

func (b *Block) Hash() []byte {
	copyBlock := *b
	copyBlock.CurrentBlockHash = nil // Tránh tự tham chiếu khi băm
	data, _ := json.Marshal(copyBlock)
	hash := sha256.Sum256(data)
	return hash[:]
}

func ValidateBlock(block *Block, prevBlock *Block) bool {
	// 1. Check previous hash (skip for genesis block)
	if prevBlock != nil && !bytes.Equal(block.PreviousBlockHash, prevBlock.CurrentBlockHash) {
		fmt.Println("❌ Invalid previous hash")
		return false
	}

	// 2. Validate each transaction's signature
	for _, tx := range block.Transactions {
		pubKey, err := cryptohelper.BytesToPublicKey(tx.PublicKey)
		if err != nil {
			fmt.Println("❌ Invalid public key in tx")
			return false
		}
		if !VerifyTransaction(tx, pubKey) {
			fmt.Println("❌ Invalid signature in tx")
			return false
		}
	}

	// 3. Rebuild MPT and check root
	var txHashes [][]byte
	for _, tx := range block.Transactions {
		txHashes = append(txHashes, tx.Hash())
	}
	_, computedRoot := mpt.BuildMPTFromTxHashes(txHashes)
	if !bytes.Equal(block.MerkleRoot, computedRoot) {
		fmt.Println("❌ Invalid MPT Merkle root")
		return false
	}

	return true
}

func ProtoToBlock(pb *nodepb.Block) *Block {
	var txs []*Transaction
	for _, ptx := range pb.Transactions {
		tx := &Transaction{
			Sender:    ptx.Sender,
			Receiver:  ptx.Receiver,
			Amount:    ptx.Amount,
			Timestamp: ptx.Timestamp,
			Signature: ptx.Signature,
			PublicKey: ptx.PublicKey,
		}
		txs = append(txs, tx)
	}

	return &Block{
		Height:            pb.Height,
		Transactions:      txs,
		MerkleRoot:        pb.MerkleRoot,
		PreviousBlockHash: pb.PreviousBlockHash,
		CurrentBlockHash:  pb.CurrentBlockHash,
		Timestamp:         pb.Timestamp,
	}
}

func BlockToProto(b *Block) *nodepb.Block {
	var ptxs []*nodepb.Transaction
	for _, tx := range b.Transactions {
		ptx := &nodepb.Transaction{
			Sender:    tx.Sender,
			Receiver:  tx.Receiver,
			Amount:    tx.Amount,
			Timestamp: tx.Timestamp,
			Signature: tx.Signature,
			PublicKey: tx.PublicKey,
		}
		ptxs = append(ptxs, ptx)
	}

	return &nodepb.Block{
		Height:            b.Height,
		Transactions:      ptxs,
		MerkleRoot:        b.MerkleRoot,
		PreviousBlockHash: b.PreviousBlockHash,
		CurrentBlockHash:  b.CurrentBlockHash,
		Timestamp:         b.Timestamp,
	}
}
