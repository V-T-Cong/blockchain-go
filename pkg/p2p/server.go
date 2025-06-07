package p2p

import (
	"context"
	"fmt"
	"log"

	"blockchain-go/pkg/blockchain"
	"blockchain-go/pkg/wallet"
	"blockchain-go/proto/nodepb"
)

type NodeServer struct {
	nodepb.UnimplementedNodeServiceServer
	PendingTxs []*blockchain.Transaction
}

func (s *NodeServer) SendTransaction(ctx context.Context, tx *nodepb.Transaction) (*nodepb.Status, error) {
	fmt.Println("📩 Received transaction")

	txInternal := &blockchain.Transaction{
		Sender:    tx.Sender,
		Receiver:  tx.Receiver,
		Amount:    tx.Amount,
		Timestamp: tx.Timestamp,
		Signature: tx.Signature,
		PublicKey: tx.PublicKey,
	}

	// fmt.Printf("🔐 PublicKey length (server): %d\n", len(tx.PublicKey))
	// fmt.Printf("🧪 Received PublicKey: %x\n", tx.PublicKey)
	pubKey, err := wallet.BytesToPublicKey(tx.PublicKey)
	if err != nil {
		log.Printf("Failed to parse public key: %v", err)
		return &nodepb.Status{Success: false, Message: "Invalid public key"}, nil
	}

	if !wallet.VerifyTransaction(txInternal, pubKey) {
		log.Println("❌ Invalid signature")
		return &nodepb.Status{Message: "Invalid signature", Success: false}, nil
	}

	s.PendingTxs = append(s.PendingTxs, txInternal)
	log.Printf("✅ Tx added. Total pending: %d\n", len(s.PendingTxs))

	return &nodepb.Status{Message: "Transaction received", Success: true}, nil
}
