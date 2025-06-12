package main

import (
	"blockchain-go/pkg/blockchain"
	"blockchain-go/pkg/wallet"
	"blockchain-go/proto/nodepb"
	"context"
	// "fmt"
	"google.golang.org/grpc"
	"log"
	"time"
)

func main() {

	aliceKey, _ := wallet.GenerateKeyPair()
	bobKey, _ := wallet.GenerateKeyPair()

	aAddr := wallet.PublicKeyToAddress(&aliceKey.PublicKey)
	bAddr := wallet.PublicKeyToAddress(&bobKey.PublicKey)

	// open connection
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())

	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := nodepb.NewNodeServiceClient(conn)

	amounts := []float64{500.0, 250.0}

	for i, amt := range amounts {
		tx := &blockchain.Transaction{
			Sender:    aAddr,
			Receiver:  bAddr,
			Amount:    amt,
			Timestamp: time.Now().Unix(),
		}

		err := wallet.SignTransaction(tx, aliceKey)
		if err != nil {
			log.Fatalf("Failed to sign transaction %d: %v", i+1, err)
		}

		// convert transaction to protobuf
		txProto := &nodepb.Transaction{
			Sender:    tx.Sender,
			Receiver:  tx.Receiver,
			Amount:    tx.Amount,
			Timestamp: tx.Timestamp,
			Signature: tx.Signature,
			PublicKey: tx.PublicKey,
		}

		// fmt.Printf("🔐 PublicKey length (client): %d\n", len(tx.PublicKey))
		res, err := client.SendTransaction(context.Background(), txProto)
		if err != nil {
			log.Fatalf("SendTransaction failed: %v", err)
		}

		log.Printf("✅ Response: %s (success: %v)", res.Message, res.Success)
	}
}
