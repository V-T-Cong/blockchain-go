package p2p

import (
	"context"
	"log"
	"time"

	"blockchain-go/blockchain/pkg/p2p/nodepb"
	"google.golang.org/grpc"
)

func SendTransactionToPeer(addr string, tx *nodepb.Transaction) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(3*time.Second))
	if err != nil {
		log.Printf("❌ Connect failed: %v\n", err)
		return
	}
	defer conn.Close()
	client := nodepb.NewNodeServiceClient(conn)
	resp, err := client.SendTransaction(context.Background(), tx)
	if err != nil {
		log.Printf("❌ SendTransaction error: %v", err)
		return
	}
	log.Println("✅ Response:", resp.Message)
}
