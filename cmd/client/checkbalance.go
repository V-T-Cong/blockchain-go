package main

import (
	"blockchain-go/proto/nodepb"
	"context"
	"encoding/hex"
	"flag"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Sử dụng flag để nhận địa chỉ ví từ dòng lệnh
	// Ví dụ: go run ./cmd/client/checkbalance.go --address=...
	address := flag.String("address", "", "The wallet address to check balance for")
	flag.Parse()

	if *address == "" {
		log.Fatalf("❌ Please provide a wallet address using the --address flag.")
	}

	// Chuyển địa chỉ từ dạng chuỗi hex sang dạng bytes
	addrBytes, err := hex.DecodeString(*address)
	if err != nil {
		log.Fatalf("❌ Invalid address format. Please provide a valid hex string. Error: %v", err)
	}

	// Mở kết nối đến node leader
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Tạo một client cho NodeService
	client := nodepb.NewNodeServiceClient(conn)

	// Tạo request với địa chỉ đã được chuyển đổi
	req := &nodepb.GetBalanceRequest{
		Address: addrBytes,
	}

	log.Printf("🔍 Checking balance for address: %s", *address)

	// Gọi đến gRPC endpoint GetBalance
	res, err := client.GetBalance(context.Background(), req)
	if err != nil {
		log.Fatalf("❌ Failed to get balance: %v", err)
	}

	// In kết quả ra màn hình
	log.Printf("======================================")
	log.Printf("👤 Address: %s", *address)
	log.Printf("💰 Balance: %.2f", res.Balance)
	log.Printf("🔢 Nonce:   %d", res.Nonce)
	log.Printf("======================================")
}
