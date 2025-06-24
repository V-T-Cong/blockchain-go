package main

import (
	"blockchain-go/pkg/blockchain"
	"blockchain-go/pkg/consensus"
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
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	totalNodes := len(peerAddrs) + 1

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

	networkAdapter := p2p_v2.NewGrpcAdapter(leaderAddr, peerAddrs)

	consensusManager := consensus.NewManager(nodeID, totalNodes, db, stateManager, latestBlock, networkAdapter)

	// === Tạo server node ===
	server := &p2p_v2.NodeServer{
		NodeID:     nodeID,
		IsLeader:   isLeader,
		Consensus:  consensusManager,
		State:      stateManager,
		PendingTxs: []*blockchain.Transaction{},
	}

	if !isLeader {
		syncFromLeader(leaderAddr, db, stateManager, consensusManager)

		latestBlock, err = db.GetLatestBlock()
		if err != nil {
			log.Fatalf("❌ Failed to get latest block after sync: %v", err)
		}
		server.Consensus.LatestBlock = latestBlock
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

func syncFromLeader(leaderAddr string, db *storage.DB, stateManager *state.State, consensusManager *consensus.Manager) {
	log.Println("🔄 Syncing blocks from leader...")
	var latestBlock, _ = db.GetLatestBlock()

	startHeight := 0

	if latestBlock != nil {
		startHeight = int(latestBlock.Height) + 1
	}

	var conn *grpc.ClientConn
	var err error

	for attempt := 1; attempt <= 5; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		conn, err = grpc.DialContext(ctx, leaderAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
		if err == nil {
			break
		}
		log.Printf("⏳ [Attempt %d] Waiting for leader at %s...", attempt, leaderAddr)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		log.Fatalf("❌ Could not connect to leader after all retries: %v", err)
	}
	defer conn.Close()

	client := nodepb.NewNodeServiceClient(conn)
	res, err := client.GetBlockFromHeight(context.Background(), &nodepb.HeightRequest{FromHeight: int64(startHeight)})
	if err != nil {
		log.Fatalf("❌ Sync failed during GetBlockFromHeight: %v", err)
	}

	if len(res.Blocks) == 0 {
		log.Printf("✅ Sync done. Already at latest height %d.", startHeight-1)
		return
	}

	log.Printf("⛓️  Received %d blocks from leader. Applying...", len(res.Blocks))
	for _, pb := range res.Blocks {
		block := blockchain.ProtoToBlock(pb)
		// Sử dụng trực tiếp ConsensusManager để commit block,
		// việc này đảm bảo tính nhất quán vì nó cũng xác thực lại block.
		if err := consensusManager.CommitBlock(block); err != nil {
			log.Fatalf("❌ Failed to commit synced block %d: %v", block.Height, err)
		}
		log.Printf("⛓️  Synced and committed block at height %d", block.Height)
	}
}
