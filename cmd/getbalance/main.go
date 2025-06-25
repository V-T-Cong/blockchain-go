package main

import (
	"blockchain-go/proto/nodepb"
	"context"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"time"
)

func main() {
	address := flag.String("address", "", "The address to check the balance of")
	flag.Parse()

	if *address == "" {
		log.Fatal("❌ Address is required. Use --address=<your_address>")
	}

	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := nodepb.NewNodeServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := client.GetBalance(ctx, &nodepb.GetBalanceRequest{Address: *address})
	if err != nil {
		log.Fatalf("❌ Could not get balance: %v", err)
	}

	fmt.Println("--- Account Balance ---")
	fmt.Printf("🏦 Address: %s\n", res.Address)
	fmt.Printf("💰 Balance: %f\n", res.Balance)
	fmt.Println("-----------------------")
}
