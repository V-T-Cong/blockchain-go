package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"blockchain-go/pkg/blockchain"
	"blockchain-go/pkg/storage"
	"blockchain-go/pkg/wallet"
)

func main() {
	// Setup DB directory
	dbPath := "../../data/testdb"
	_ = os.MkdirAll(dbPath, os.ModePerm)

	db, err := storage.OpenDB(dbPath)
	if err != nil {
		log.Fatal("❌ Failed to open DB:", err)
	}
	defer db.Close()

	// Create wallets
	aliceKey, _ := wallet.GenerateKeyPair()
	bobKey, _ := wallet.GenerateKeyPair()
	aAddr := wallet.PublicKeyToAddress(&aliceKey.PublicKey)
	bAddr := wallet.PublicKeyToAddress(&bobKey.PublicKey)

	var prevHash []byte = []byte("GENESIS")

	// Create 4 blocks
	for i := 1; i <= 4; i++ {
		amount := float64(10 * i)

		var senderAddr, receiverAddr []byte
		var senderKey *ecdsa.PrivateKey

		if i%2 == 0 {
			senderAddr = bAddr
			receiverAddr = aAddr
			senderKey = bobKey
		} else {
			senderAddr = aAddr
			receiverAddr = bAddr
			senderKey = aliceKey
		}

		// Create and sign transaction
		tx := blockchain.NewTransaction(senderAddr, receiverAddr, amount)
		wallet.SignTransaction(tx, senderKey)

		// ⚠️ FIX: Thêm height = i
		block := blockchain.NewBlock([]*blockchain.Transaction{tx}, prevHash, i)

		err := db.SaveBlock(block)
		if err != nil {
			log.Fatalf("❌ Failed to save block %d: %v", i, err)
		}
		fmt.Printf("✅ Block %d saved. Hash: %x\n", i, block.CurrentBlockHash)

		// Update prevHash
		prevHash = block.CurrentBlockHash
	}

	// Retrieve and print the last block
	latestBlock, err := db.GetLatestBlock()
	if err != nil {
		log.Fatal("❌ Failed to get latest block:", err)
	}

	fmt.Printf("⛓️ Latest Block Hash: %x\n", latestBlock.CurrentBlockHash)
	fmt.Printf("\n📦 Last Block Info:\n")
	fmt.Printf("🔑 Hash: %x\n", latestBlock.CurrentBlockHash)
	fmt.Printf("🌲 Merkle Root: %x\n", latestBlock.MerkleRoot)
	fmt.Printf("📜 Tx Count: %d\n", len(latestBlock.Transactions))
	for i, tx := range latestBlock.Transactions {
		fmt.Printf("🧾 Tx %d - From: %s To: %s Amount: %.2f\n",
			i+1,
			hex.EncodeToString(tx.Sender),
			hex.EncodeToString(tx.Receiver),
			tx.Amount,
		)
	}
}
