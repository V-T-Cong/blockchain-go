package main

import (
	"blockchain-go/pkg/blockchain"
	"blockchain-go/pkg/storage"
	"blockchain-go/pkg/wallet"
	"fmt"
	"log"
	"os"
)

func main() {
	alicePriv, _ := wallet.GenerateKeyPair()
	fmt.Println("Alice's Private Key:", alicePriv)
	bobPriv, _ := wallet.GenerateKeyPair()
	fmt.Println("Bob's Private Key:", bobPriv)

	aliceAddr := wallet.PublicKeyToAddress(&alicePriv.PublicKey)
	fmt.Println("Alice's Address:", aliceAddr)
	bobAddr := wallet.PublicKeyToAddress(&bobPriv.PublicKey)
	fmt.Println("Bob's Address:", bobAddr)

	tx1 := blockchain.NewTransaction(aliceAddr, bobAddr, 100)
	wallet.SignTransaction(tx1, alicePriv)

	tx2 := blockchain.NewTransaction(bobAddr, aliceAddr, 50)
	wallet.SignTransaction(tx2, alicePriv)

	block := blockchain.NewBlock([]*blockchain.Transaction{tx1, tx2}, []byte("prevhash"))

	fmt.Printf("📦 Block Hash: %x\n", block.CurrentBlockHash)
	fmt.Printf("🌳 Merkle Root: %x\n", block.MerkleRoot)

	// Tamper test
	tx2.Amount = 1000
	fmt.Printf("🧪 Tampered Merkle Root: %x\n", blockchain.ComputeMerkleRoot([][]byte{tx1.Hash(), tx2.Hash()}))

	// Setup test data
	// alice, _ := wallet.GenerateKeyPair()
	// bob, _ := wallet.GenerateKeyPair()

	// tx := blockchain.NewTransaction(
	// 	wallet.PublicKeyToAddress(&alice.PublicKey),
	// 	wallet.PublicKeyToAddress(&bob.PublicKey),
	// 	50,
	// )
	// wallet.SignTransaction(tx, alice)

	// block := blockchain.NewBlock([]*blockchain.Transaction{tx}, []byte("GENESIS"))

	// // Open DB
	// dbPath := "./data/testdb"
	// _ = os.MkdirAll(dbPath, os.ModePerm)
	// db, err := storage.OpenDB(dbPath)
	// if err != nil {
	// 	log.Fatal("❌ Failed to open DB:", err)
	// }
	// defer db.Close()

	// // Save block
	// err = db.SaveBlock(block)
	// if err != nil {
	// 	log.Fatal("❌ Failed to save block:", err)
	// }
	// fmt.Println("✅ Block saved.")

	// // Load block
	// loadedBlock, err := db.GetBlock(block.CurrentBlockHash)
	// if err != nil {
	// 	log.Fatal("❌ Failed to get block:", err)
	// }
	// fmt.Printf("📦 Loaded Block Hash: %x\n", loadedBlock.CurrentBlockHash)
}
