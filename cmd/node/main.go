package main

import (
	"blockchain-go/pkg/blockchain"
	"blockchain-go/pkg/mpt"
	"blockchain-go/pkg/p2p_v2"
	"blockchain-go/pkg/storage"
	"blockchain-go/proto/nodepb"
	"context"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
)

func ctx() context.Context {
	return context.Background()
}

// Hàm này sẽ đọc tất cả các block từ DB và xây dựng lại State Trie
func rebuildStateTrieFromDB(db *storage.DB) (*mpt.MPT, *blockchain.Block, error) {
	stateTrie := mpt.NewMPT()
	latestBlock, err := db.GetLatestBlock()
	if err != nil { // Lỗi có thể là "not found", nghĩa là DB mới và chưa có block nào
		return stateTrie, nil, nil
	}

	log.Printf("Found existing data. Rebuilding state from DB up to height %d...", latestBlock.Height)

	// Chạy từ block genesis (height 0) đến block cuối cùng
	for h := 0; h <= int(latestBlock.Height); h++ {
		block, err := db.GetBlockByHeight(h)
		if err != nil {
			return nil, nil, err // Lỗi nghiêm trọng, không thể thiếu block ở giữa
		}
		// "Tua lại" các giao dịch trong từng block để cập nhật State Trie
		for _, tx := range block.Transactions {
			senderData, _ := stateTrie.Get(tx.Sender)
			senderAccount := &blockchain.Account{Balance: 1000000} // Giả định số dư ban đầu
			if senderData != nil {
				senderAccount, _ = blockchain.DeserializeAccount(senderData)
			}

			receiverData, _ := stateTrie.Get(tx.Receiver)
			receiverAccount := &blockchain.Account{Balance: 0}
			if receiverData != nil {
				receiverAccount, _ = blockchain.DeserializeAccount(receiverData)
			}

			if senderAccount.Balance >= tx.Amount {
				senderAccount.Balance -= tx.Amount
				receiverAccount.Balance += tx.Amount
			}

			newSenderData, _ := senderAccount.Serialize()
			stateTrie.Insert(tx.Sender, newSenderData)

			newReceiverData, _ := receiverAccount.Serialize()
			stateTrie.Insert(tx.Receiver, newReceiverData)
		}
	}
	log.Printf("✅ State rebuild complete. Final StateRoot: %x", stateTrie.RootHash())
	return stateTrie, latestBlock, nil
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
	peersEnv := os.Getenv("PEERS")
	var peerAddrs []string
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

	// === Xây dựng lại State Trie từ DB khi khởi động ===
	stateTrie, latestBlock, err := rebuildStateTrieFromDB(db)
	if err != nil {
		log.Fatalf("❌ Failed to rebuild state from DB: %v", err)
	}

	if latestBlock != nil {
		log.Printf("⛓️ Resumed from block: Height %d", latestBlock.Height)
	} else {
		log.Println("🌱 Starting with genesis block and empty state trie")
	}

	// === Tạo server node với đầy đủ các trường ===
	server := &p2p_v2.NodeServer{
		NodeID:               nodeID,
		LeaderAddr:           leaderAddr,
		IsLeader:             isLeader,
		DB:                   db,
		LatestBlock:          latestBlock,
		StateTrie:            stateTrie, // Dùng StateTrie đã được xây dựng lại
		PendingTxs:           []*blockchain.Transaction{},
		PendingBlocks:        make(map[string]*blockchain.Block),
		VoteCount:            make(map[string]int),
		BlockCommitted:       make(map[string]bool),
		VoteMutex:            sync.Mutex{},
		BlockProcessingMutex: sync.Mutex{}, // Khởi tạo Mutex mới
		PeerAddrs:            peerAddrs,
		TotalNodes:           total,
	}

	// === Đồng bộ các block đã bỏ lỡ (dành cho follower) ===
	if !isLeader {
		log.Println("🔄 Follower node starting up. Checking for new blocks from the leader...")

		// Xác định xem cần bắt đầu đồng bộ từ block nào.
		// `rebuildStateTrieFromDB` đã xử lý các block có sẵn trong DB.
		startHeight := 0
		if server.LatestBlock != nil {
			startHeight = int(server.LatestBlock.Height) + 1
		}

		log.Printf("...requesting blocks from height %d onwards.", startHeight)

		var res *nodepb.BlockList
		var err error

		// Vòng lặp thử lại để cho leader có thời gian khởi động
		for attempt := 1; attempt <= 5; attempt++ {
			conn, connErr := grpc.Dial(leaderAddr, grpc.WithInsecure())
			if connErr != nil {
				log.Printf("⏳ [Attempt %d] Waiting for leader at %s... (%v)", attempt, leaderAddr, connErr)
				time.Sleep(3 * time.Second)
				continue
			}

			client := nodepb.NewNodeServiceClient(conn)
			// Yêu cầu tất cả các block kể từ chiều cao của chúng ta
			res, err = client.GetBlockFromHeight(ctx(), &nodepb.HeightRequest{FromHeight: int64(startHeight)})
			conn.Close() // Đóng kết nối ngay sau khi dùng

			if err != nil {
				log.Printf("❌ [Attempt %d] Sync failed with RPC error: %v", attempt, err)
				time.Sleep(3 * time.Second)
				continue
			}

			// Nếu nhận được phản hồi (kể cả rỗng) mà không có lỗi RPC, thoát khỏi vòng lặp
			break
		}

		// Nếu vẫn lỗi sau tất cả các lần thử, thì có vấn đề nghiêm trọng
		if err != nil {
			log.Fatalf("❌ Sync failed after all retries: %v", err)
		}

		// Xử lý các block nhận được
		if res != nil && len(res.Blocks) > 0 {
			log.Printf("⛓️ Received %d new blocks from leader. Applying now...", len(res.Blocks))
			for _, pb := range res.Blocks {
				block := blockchain.ProtoToBlock(pb)

				// Thực hành tốt là xác thực block nhận được so với block cuối cùng của chúng ta
				if !blockchain.ValidateBlock(block, server.LatestBlock) {
					log.Fatalf("❌ Received an invalid block %d during sync. Halting.", block.Height)
				}

				// 1. Lưu block mới vào DB của mình
				if err := db.SaveBlock(block); err != nil {
					log.Fatalf("❌ Failed to save synced block %d: %v", block.Height, err)
				}

				// 2. Áp dụng các giao dịch để cập nhật State Trie
				// Chúng ta dùng ProcessAndUpdateState vì đây là các block đã được commit
				if _, err := server.ProcessAndUpdateState(block.Transactions); err != nil {
					log.Fatalf("❌ Failed to update state for synced block %d: %v", block.Height, err)
				}

				// 3. Cập nhật con trỏ `LatestBlock` trong bộ nhớ của server
				server.LatestBlock = block
				log.Printf("✅ Synced and applied block %d", block.Height)
			}
			log.Printf("✅ Sync and state update complete. Node is now at height %d.", server.LatestBlock.Height)
		} else {
			log.Printf("✅ Node is already up-to-date with the leader.")
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
