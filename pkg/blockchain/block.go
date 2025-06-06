package blockchain

import (
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

	fmt.Printf("txHashes: %x\n", txHashes)

	merkleRoot := ComputeMerkleRoot(txHashes)

	block := &Block{
		Height:            int64(height),
		Transactions:      transactions,
		MerkleRoot:        merkleRoot,
		PreviousBlockHash: previousBlockHash,
		Timestamp:         time.Now().Unix(),
	}

	block.CurrentBlockHash = block.Hash()
	return block
}

func (b *Block) Hash() []byte {
	copyBlock := *b
	copyBlock.CurrentBlockHash = nil // avoid self-inclusion
	data, _ := json.Marshal(copyBlock)
	hash := sha256.Sum256(data)
	return hash[:]
}
