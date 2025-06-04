package blockchain

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

type Block struct {
	Transactions      []*Transaction
	MerkleRoot        []byte
	PreviousBlockHash []byte
	CurrentBlockHash  []byte
}

func NewBlock(transactions []*Transaction, previousHash []byte) *Block {
	var txHashes [][]byte
	for _, tx := range transactions {
		txHashes = append(txHashes, tx.Hash())
	}

	fmt.Printf("txHashes: %x\n", txHashes)

	merkleRoot := ComputeMerkleRoot(txHashes)

	block := &Block{
		Transactions:      transactions,
		MerkleRoot:        merkleRoot,
		PreviousBlockHash: previousHash,
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
