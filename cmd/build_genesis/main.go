package main

import (
	"blockchain-go/pkg/blockchain"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
)

type GenesisData struct {
	Alloc map[string]struct {
		Balance float64 `json:"balance"`
	} `json:"alloc"`
}

func main() {
	genesisFile, err := os.ReadFile("genesis.json")
	if err != nil {
		panic(fmt.Sprintf("Failed to read genesis.json: %v", err))
	}

	var genesisData GenesisData
	if err := json.Unmarshal(genesisFile, &genesisData); err != nil {
		panic(fmt.Sprintf("Failed to parse genesis.json: %v", err))
	}

	var transactions []*blockchain.Transaction
	for addr, data := range genesisData.Alloc {
		receiverAddr, err := hex.DecodeString(addr)
		if err != nil {
			panic(fmt.Sprintf("Invalid address in genesis.json: %s", addr))
		}
		tx := &blockchain.Transaction{
			Sender: []byte("GENESIS"), Receiver: receiverAddr, Amount: data.Balance,
			Timestamp: 0, Signature: nil, PublicKey: nil,
		}
		transactions = append(transactions, tx)
	}

	// Gọi hàm NewBlock sạch
	genesisBlock := blockchain.NewBlock(transactions, []byte{}, 0)

	blockData, err := json.MarshalIndent(genesisBlock, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal genesis block: %v", err))
	}

	if err := os.WriteFile("genesis.dat", blockData, 0644); err != nil {
		panic(fmt.Sprintf("Failed to write genesis.dat: %v", err))
	}

	fmt.Println("✅ Genesis block 'genesis.dat' created successfully!")
	fmt.Printf("   - Hash: %x\n", genesisBlock.CurrentBlockHash)
	fmt.Printf("   - Merkle Root: %x\n", genesisBlock.MerkleRoot)
	fmt.Printf("   - Transactions: %d\n", len(genesisBlock.Transactions))
}
