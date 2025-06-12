package main

import (
	"blockchain-go/pkg/mpt"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func hashTx(data string) []byte {
	h := sha256.Sum256([]byte(data))
	return h[:]
}

func main() {
	trie := mpt.NewMPT()

	txs := []string{"tx1", "tx2", "tx3"}
	for _, tx := range txs {
		h := hashTx(tx)
		trie.Insert(h, h)
	}

	fmt.Println("Root Hash:", hex.EncodeToString(trie.RootHash()))

	// Generate proof for tx2
	txHash := hashTx("tx2")
	proof := trie.GenerateProof(txHash)

	fmt.Println("Proof for tx2:")
	for _, p := range proof {
		fmt.Println(hex.EncodeToString(p))
	}
}
