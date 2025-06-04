package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"blockchain-go/blockchain/pkg/p2p/nodepb"
	"blockchain-go/pkg/wallet"

	"google.golang.org/grpc"
)

func main() {
	// Generate Alice and Bob wallets
	alice, _ := wallet.GenerateKeyPair()
	bob, _ := wallet.GenerateKeyPair()

	// Create transaction from Alice to Bob
	tx := &nodepb.Transaction{
		Sender:    wallet.PublicKeyToAddress(&alice.PublicKey),
		Receiver:  wallet.PublicKeyToAddress(&bob.PublicKey),
		Amount:    42.0,
		Timestamp: time.Now().Unix(),
	}

	// Sign the transaction
	txHash := wallet.HashTransactionFields(tx)
	r, s, _ := wallet.SignHash(txHash, alice)
	tx.Signature = append(r.Bytes(), s.Bytes()...)

	// Connect to running gRPC server (node1)
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(5*time.Second))
	if err != nil {
		log.Fatalf("❌ Could not connect: %v", err)
	}
	defer conn.Close()

	client := nodepb.NewNodeServiceClient(conn)
	res, err := client.SendTransaction(context.Background(), tx)
	if err != nil {
		log.Fatalf("❌ SendTransaction failed: %v", err)
	}

	fmt.Println("✅ Server Response:", res.Message)
}
