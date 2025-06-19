package main

import (
	"blockchain-go/pkg/blockchain"
	"blockchain-go/pkg/p2p_v2"
	"blockchain-go/pkg/state"
	"blockchain-go/pkg/storage"
	"blockchain-go/proto/nodepb"
	"encoding/json"
	"errors"

	"context"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"google.golang.org/grpc"
)

func ctx() context.Context {
	return context.Background()
}

func main() {
	// === Cấu hình từ biến môi trường ===
	nodeID := os.Getenv("NODE_ID")
	leaderAddr := os.Getenv("LEADER_ADDR")
	if leaderAddr == "" {
		log.Fatal("❌ LEADER_ADDR is not set")
	}
	isLeaderEnv := os.Getenv("IS_LEADER")
	isLeader := strings.ToLower(isLeaderEnv) == "true"
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

	// =============

	// === APPLY GENESIS BLOCK ===
	_, err = db.GetLatestBlock()
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			log.Printf("🌱 Node %s: Database is empty. Loading genesis block...", nodeID)
			genesisData, err := os.ReadFile("genesis.dat")
			if err != nil {
				log.Fatalf("❌ Could not read genesis.dat: %v.", err)
			}
			var genesisBlock blockchain.Block
			if err := json.Unmarshal(genesisData, &genesisBlock); err != nil {
				log.Fatalf("❌ Failed to parse genesis block: %v", err)
			}
			if err := db.SaveBlock(&genesisBlock); err != nil {
				log.Fatalf("❌ Failed to save genesis block to DB: %v", err)
			}
			log.Printf("✅ Node %s: Genesis block loaded and saved to DB.", nodeID)
		} else {
			log.Fatalf("❌ Error checking for latest block: %v", err)
		}
	}

	// === Khởi tạo State Manager ===
	stateManager, err := state.NewState(db)
	if err != nil {
		log.Fatalf("❌ Failed to initialize state manager: %v", err)
	}

	// === Xây dựng lại trạng thái từ blockchain ===
	if err := stateManager.RebuildStateFromBlockchain(); err != nil {
		log.Fatalf("❌ Failed to rebuild state: %v", err)
	}

	// === Lấy block cuối cùng nếu có ===
	latestBlock, _ := db.GetLatestBlock()
	if latestBlock != nil {
		log.Printf("⛓️ Latest block: Height %d", latestBlock.Height)
	} else {
		log.Println("🌱 Starting with genesis block")
	}

	// === Tạo server node ===
	server := &p2p_v2.NodeServer{
		NodeID:         nodeID,
		LeaderAddr:     leaderAddr,
		IsLeader:       isLeader,
		DB:             db,
		State:          stateManager,
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

		startHeight := 0
		if latestBlock != nil {
			startHeight = int(latestBlock.Height) + 1
		}

		var res *nodepb.BlockList
		var err error

		for attempt := 1; attempt <= 5; attempt++ {
			conn, connErr := grpc.Dial(leaderAddr, grpc.WithInsecure())
			if connErr != nil {
				log.Printf("⏳ [Attempt %d] Waiting for leader at %s...", attempt, leaderAddr)
				time.Sleep(2 * time.Second)
				continue
			}
			defer conn.Close()

			client := nodepb.NewNodeServiceClient(conn)
			res, err = client.GetBlockFromHeight(ctx(), &nodepb.HeightRequest{FromHeight: int64(startHeight)})

			if err != nil {
				log.Printf("❌ [Attempt %d] Sync failed: %v", attempt, err)
				time.Sleep(2 * time.Second)
				continue
			}

			break // thành công
		}

		if err != nil {
			log.Fatalf("❌ Sync failed after retries: %v", err)
		}

		if len(res.Blocks) == 0 {
			log.Printf("✅ Sync done at height %d (no new blocks)", startHeight-1)
		} else {
			log.Printf("⛓️ Received %d blocks from leader. Applying...", len(res.Blocks))
			for _, pb := range res.Blocks {
				block := blockchain.ProtoToBlock(pb)
				// Lưu block vào DB của follower
				if err := db.SaveBlock(block); err != nil {
					log.Fatalf("❌ Failed to save synced block: %v", err)
				}

				// CẬP NHẬT STATE CỦA FOLLOWER - ĐÂY LÀ PHẦN THIẾU
				for _, tx := range block.Transactions {
					if err := stateManager.ApplyTransaction(tx); err != nil {
						log.Printf("❌ Failed to apply transaction from synced block %d: %v", block.Height, err)
					}
				}
				log.Printf("⛓️ Synced and applied block at height %d", block.Height)
			}
			latestBlock, _ = db.GetLatestBlock()
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
