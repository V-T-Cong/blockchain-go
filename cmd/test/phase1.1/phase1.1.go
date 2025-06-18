package main

import (
	"blockchain-go/pkg/blockchain"
	"blockchain-go/pkg/wallet"
	"fmt"
)

func main() {

	alicePriv, _ := wallet.GenerateKeyPair()
	fmt.Printf("Alice private key: %v\n", alicePriv)
	fmt.Printf("Alice public key: %v\n", alicePriv.PublicKey)
	bobPriv, _ := wallet.GenerateKeyPair()
	fmt.Printf("Bob private Key: %v\n", bobPriv)
	fmt.Printf("Bob public Key: %v\n", bobPriv.PublicKey)

	aliceAddr := wallet.PublicKeyToAddress(&alicePriv.PublicKey)
	fmt.Printf("Alice address:: %v\n", aliceAddr)
	bobAddr := wallet.PublicKeyToAddress(&bobPriv.PublicKey)
	fmt.Printf("Bob address:: %v\n", bobAddr)

	tx := blockchain.NewTransaction(aliceAddr, bobAddr, 100)
	fmt.Printf("Transactions:: %v\n", tx)
	wallet.SignTransaction(tx, alicePriv)

	if blockchain.VerifyTransaction(tx, &alicePriv.PublicKey) {
		fmt.Println("✅ Transaction from Alice to Bob is valid!")
	} else {
		fmt.Println("❌ Invalid transaction!")
	}
}
