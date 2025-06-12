package main

import (
	"blockchain-go/pkg/blockchain"
	"blockchain-go/pkg/p2p"
	"blockchain-go/pkg/storage"
	"blockchain-go/proto/nodepb"
	"context"

	"log"
	"net"
	"os"
	"strings"
	"sync"

	"google.golang.org/grpc"
)

func ctx() context.Context {
	return context.Background()
}

func main() {
	// === Cấu hình từ biến môi trường ===
	nodeID := os.Getenv("NODE_ID")
	leaderAddr := os.Getenv("LEADER_ADDR")
	peersEnv := os.Getenv("PEERS") // Ví dụ: node2:50051,node3:50051
	peerAddrs := []string{}
	if peersEnv != "" {
		peerAddrs = strings.Split(peersEnv, ",")
	}

	total := len(peerAddrs) + 1
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
		TotalNodes:     total,
	}

	if nodeID != "node1" {
		log.Println("🔄 Syncing blocks from leader...")

		currentHeight := int64(-1)
		if latestBlock != nil {
			currentHeight = latestBlock.Height
		}

		conn, err := grpc.Dial(leaderAddr, grpc.WithInsecure())
		if err != nil {
			log.Fatalf("❌ Cannot connect to leader: %v", err)
		}
		defer conn.Close()

		client := nodepb.NewNodeServiceClient(conn)

		for {
			req := &nodepb.BlockRequest{Height: currentHeight + 1}
			pb, err := client.GetBlock(ctx(), req)
			if err != nil {
				log.Printf("✅ Sync done at height %d", currentHeight)
				break
			}

			block := blockchain.ProtoToBlock(pb)
			if err := db.SaveBlock(block); err != nil {
				log.Fatalf("❌ Failed to save synced block: %v", err)
			}
			currentHeight++
			log.Printf("⛓️ Synced block %d", currentHeight)
		}
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
