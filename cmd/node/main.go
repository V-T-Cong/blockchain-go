package main

import (
	"blockchain-go/pkg/blockchain"
	"blockchain-go/pkg/p2p"
	"blockchain-go/pkg/storage"
	"blockchain-go/proto/nodepb"

	"log"
	"net"
	"os"
	"strings"
	"sync"

	"google.golang.org/grpc"
)

func main() {
	// === Cấu hình từ biến môi trường ===
	nodeID := os.Getenv("NODE_ID")
	leaderAddr := os.Getenv("LEADER_ADDR")
	peersEnv := os.Getenv("PEERS") // Ví dụ: node2:50051,node3:50051
	peerAddrs := []string{}
	if peersEnv != "" {
		peerAddrs = strings.Split(peersEnv, ",")
	}

	// === Khởi tạo DB ===
	dbPath := "data/" + nodeID
	if err := os.MkdirAll(dbPath, os.ModePerm); err != nil {
		log.Fatalf("❌ Failed to create data directory: %v", err)
	}
	db, err := storage.OpenDB(dbPath)
	if err != nil {
		log.Fatalf("❌ Failed to open DB: %v", err)
	}
	defer db.Close()

	// === Lấy block cuối cùng nếu có ===
	latestBlock, _ := db.GetLatestBlock()
	if latestBlock != nil {
		log.Printf("⛓️ Latest block: Height %d", latestBlock.Height)
	} else {
		log.Println("🌱 Starting with genesis block")
	}

	// === Tạo server node ===
	server := &p2p.NodeServer{
		NodeID:         nodeID,
		LeaderAddr:     leaderAddr,
		DB:             db,
		LatestBlock:    latestBlock,
		PendingTxs:     []*blockchain.Transaction{},
		PendingBlocks:  make(map[string]*blockchain.Block),
		VoteCount:      make(map[string]int),
		BlockCommitted: make(map[string]bool),
		VoteMutex:      sync.Mutex{},
		PeerAddrs:      peerAddrs,
	}

	// === Khởi động gRPC ===
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("❌ Failed to listen on :50051: %v", err)
	}

	grpcServer := grpc.NewServer()
	nodepb.RegisterNodeServiceServer(grpcServer, server)

	log.Printf("🚀 Node %s started on :50051", nodeID)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("❌ gRPC server error: %v", err)
	}
}
