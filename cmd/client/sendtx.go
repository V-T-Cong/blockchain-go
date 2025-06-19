// cmd/client/sendtx.go
package main

import (
	"blockchain-go/pkg/blockchain"
	"blockchain-go/pkg/wallet"
	"blockchain-go/proto/nodepb"
	"context"
	"encoding/hex"
	"google.golang.org/grpc"
	"log"
	"time"
)

func main() {

	log.Println("🔑 Loading wallets from files...")

	aliceWallet, err := wallet.LoadWallet("wallets/alice.json")
	if err != nil {
		log.Fatalf("❌ Failed to load alice's wallet. Did you create it first? Error: %v", err)
	}

	bobWallet, err := wallet.LoadWallet("wallets/bob.json")
	if err != nil {
		log.Fatalf("❌ Failed to load bob's wallet. Did you create it first? Error: %v", err)
	}
	log.Printf("✅ Wallets loaded. Alice's address: %s", aliceWallet.Address)

	// 2. Chuyển đổi địa chỉ từ string (hex) sang []byte để sử dụng trong giao dịch
	senderAddrBytes, err := hex.DecodeString(aliceWallet.Address)
	if err != nil {
		log.Fatalf("Failed to decode sender address: %v", err)
	}

	receiverAddrBytes, err := hex.DecodeString(bobWallet.Address)
	if err != nil {
		log.Fatalf("Failed to decode receiver address: %v", err)
	}

	// Mở kết nối đến node gRPC
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := nodepb.NewNodeServiceClient(conn)

	// Gửi một vài giao dịch
	amounts := []float64{3500.0, 5500.123}

	for i, amt := range amounts {
		log.Printf("----------------------------------")
		log.Printf("🚀 Preparing transaction #%d: %.2f coins from Alice to Bob", i+1, amt)

		tx := &blockchain.Transaction{
			Sender:    senderAddrBytes,   // Sử dụng địa chỉ đã được decode
			Receiver:  receiverAddrBytes, // Sử dụng địa chỉ đã được decode
			Amount:    amt,
			Timestamp: time.Now().Unix(),
		}

		// 3. Ký giao dịch bằng Private Key đã được nạp từ file của Alice
		err := wallet.SignTransaction(tx, aliceWallet.PrivateKey)
		if err != nil {
			log.Fatalf("Failed to sign transaction %d: %v", i+1, err)
		}

		// Chuyển đổi giao dịch sang định dạng protobuf
		txProto := &nodepb.Transaction{
			Sender:    tx.Sender,
			Receiver:  tx.Receiver,
			Amount:    tx.Amount,
			Timestamp: tx.Timestamp,
			Signature: tx.Signature,
			PublicKey: tx.PublicKey,
		}

		// Gửi giao dịch đến node
		res, err := client.SendTransaction(context.Background(), txProto)
		if err != nil {
			log.Fatalf("SendTransaction failed: %v", err)
		}

		log.Printf("✅ Response from node: %s (Success: %v)", res.Message, res.Success)
		time.Sleep(1 * time.Second) // Đợi một chút giữa các giao dịch
	}
	log.Printf("----------------------------------")
}
