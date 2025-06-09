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

	tx := &blockchain.Transaction{
		Sender:    aAddr,
		Receiver:  bAddr,
		Amount:    500.0,
		Timestamp: time.Now().Unix(),
	}

	err := wallet.SignTransaction(tx, aliceKey)
	if err != nil {
		log.Fatalf("Failed to sign: %v", err)
	}

	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())

	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := nodepb.NewNodeServiceClient(conn)

	// 5. Chuyển transaction thành protobuf
	txProto := &nodepb.Transaction{
		Sender:    tx.Sender,
		Receiver:  tx.Receiver,
		Amount:    tx.Amount,
		Timestamp: tx.Timestamp,
		Signature: tx.Signature,
		PublicKey: tx.PublicKey,
	}

	// fmt.Printf("🔐 PublicKey length (client): %d\n", len(tx.PublicKey))

	// 6. Gửi giao dịch
	res, err := client.SendTransaction(context.Background(), txProto)
	if err != nil {
		log.Fatalf("SendTransaction failed: %v", err)
	}

	log.Printf("✅ Response: %s (success: %v)", res.Message, res.Success)
}
