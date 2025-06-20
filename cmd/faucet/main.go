package main

import (
	"blockchain-go/pkg/blockchain"
	"blockchain-go/pkg/wallet"
	"blockchain-go/proto/nodepb"
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"strconv"
	"time"
)

func main() {
	// 1. Định nghĩa và đọc các tham số từ dòng lệnh
	recipientAddr := flag.String("to", "", "The address of the recipient")
	amountStr := flag.String("amount", "0", "The amount to send")
	flag.Parse()

	if *recipientAddr == "" {
		log.Fatal("❌ Recipient address is required. Use --to=<address>")
	}
	amount, err := strconv.ParseFloat(*amountStr, 64)
	if err != nil || amount <= 0 {
		log.Fatal("❌ Invalid amount. Must be a positive number. Use --amount=<number>")
	}

	// 2. Nạp ví của Faucet
	log.Println("🔑 Loading faucet wallet...")
	faucetWallet, err := wallet.LoadWallet("wallets/faucet.json")
	if err != nil {
		log.Fatalf("❌ Failed to load faucet wallet. Did you create it? Error: %v", err)
	}
	log.Printf("✅ Faucet wallet loaded. Address: %s", faucetWallet.Address)

	// 3. Chuẩn bị địa chỉ cho giao dịch
	senderAddrBytes, _ := hex.DecodeString(faucetWallet.Address)
	receiverAddrBytes, err := hex.DecodeString(*recipientAddr)
	if err != nil {
		log.Fatalf("❌ Invalid recipient address format: %v", err)
	}

	// 4. Kết nối đến node Leader
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to node: %v", err)
	}
	defer conn.Close()
	client := nodepb.NewNodeServiceClient(conn)

	// 5. Tạo, ký và gửi giao dịch
	log.Printf("🚀 Preparing to send %.2f coins to %s", amount, *recipientAddr)
	tx := &blockchain.Transaction{
		Sender:    senderAddrBytes,
		Receiver:  receiverAddrBytes,
		Amount:    amount,
		Timestamp: time.Now().Unix(),
	}

	if err := wallet.SignTransaction(tx, faucetWallet.PrivateKey); err != nil {
		log.Fatalf("Failed to sign transaction: %v", err)
	}

	txProto := &nodepb.Transaction{
		Sender: tx.Sender, Receiver: tx.Receiver, Amount: tx.Amount,
		Timestamp: tx.Timestamp, Signature: tx.Signature, PublicKey: tx.PublicKey,
	}

	res, err := client.SendTransaction(context.Background(), txProto)
	if err != nil {
		log.Fatalf("SendTransaction failed: %v", err)
	}

	if res.Success {
		fmt.Println("✅ Faucet transaction sent successfully!")
	} else {
		fmt.Printf("❌ Faucet transaction failed: %s\n", res.Message)
	}
}
