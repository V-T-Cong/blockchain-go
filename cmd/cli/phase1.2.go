package main

import (
	"blockchain-go/pkg/blockchain"
	"blockchain-go/pkg/wallet"
	"fmt"
)

func main() {
	alice, _ := wallet.GenerateKeyPair()
	bob, _ := wallet.GenerateKeyPair()

	aAddr := wallet.PublicKeyToAddress(&alice.PublicKey)
	bAddr := wallet.PublicKeyToAddress(&bob.PublicKey)

	tx1 := blockchain.NewTransaction(aAddr, bAddr, 10)
	wallet.SignTransaction(tx1, alice)

	tx2 := blockchain.NewTransaction(bAddr, aAddr, 5)
	wallet.SignTransaction(tx2, bob)

	block := blockchain.NewBlock([]*blockchain.Transaction{tx1, tx2}, []byte("GENERIS"), 0)
	fmt.Printf("📦 Block Hash: %x\n", block.CurrentBlockHash)
	fmt.Printf("🌳 Merkle Root: %x\n", block.MerkleRoot)

	// Tamper test
	tx2.Amount = 1000
	fmt.Printf("🧪 Tampered Merkle Root: %x\n", blockchain.ComputeMerkleRoot([][]byte{tx1.Hash(), tx2.Hash()}))
}
