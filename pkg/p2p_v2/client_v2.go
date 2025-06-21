package p2p_v2

import (
	"blockchain-go/pkg/blockchain"
	"blockchain-go/pkg/mpt"
	"blockchain-go/pkg/storage"
	"blockchain-go/proto/nodepb"
	"context"
	"time"

	"fmt"
	"log"

	"google.golang.org/grpc"
	// "time"
)

type NodeClient struct {
	client nodepb.NodeServiceClient
}

func ProposeBlockToFollowers(block *blockchain.Block, peerAddrs []string) {
	log.Printf("📤 Proposing block to %d followers...", len(peerAddrs))

	pb := blockchain.BlockToProto(block)

	for _, addr := range peerAddrs {
		go func(peerAddr string) {
			log.Printf("🌐 Sending block to %s", peerAddr) // <-- helpful for debug

			conn, err := grpc.Dial(peerAddr, grpc.WithInsecure())
			if err != nil {
				log.Printf("❌ Cannot connect to %s: %v", peerAddr, err)
				return
			}
			defer conn.Close()

			client := nodepb.NewNodeServiceClient(conn)
			res, err := client.ProposeBlock(context.Background(), pb)
			if err != nil {
				log.Printf("❌ Failed to send block to %s: %v", peerAddr, err)
				return
			}

			if res.Success {
				log.Printf("✅ Sent block to %s: %s", peerAddr, res.Message)
			} else {
				log.Printf("⚠️ Block rejected by %s: %s", peerAddr, res.Message)
			}
		}(addr)
	}
}

func SendVoteToLeader(vote *nodepb.Vote, leaderAddr string) error {
	conn, err := grpc.Dial(leaderAddr, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("failed to connect to leader: %w", err)
	}
	defer conn.Close()

	client := nodepb.NewNodeServiceClient(conn)

	_, err = client.VoteBlock(context.Background(), vote)
	if err != nil {
		return fmt.Errorf("failed to send vote: %w", err)
	}

	return nil
}

func BroadcastCommittedBlock(block *blockchain.Block, peerAddrs []string, selfAddr string) {
	pb := blockchain.BlockToProto(block)

	for _, addr := range peerAddrs {
		if addr == selfAddr {
			continue // Tránh gửi lại chính mình
		}

		go func(peerAddr string) {
			var lastErr error
			for attempt := 1; attempt <= 3; attempt++ {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()

				conn, err := grpc.Dial(peerAddr, grpc.WithInsecure(), grpc.WithBlock())
				if err != nil {
					lastErr = err
					log.Printf("❌ [Attempt %d] Cannot connect to %s: %v", attempt, peerAddr, err)
					time.Sleep(500 * time.Millisecond)
					continue
				}

				client := nodepb.NewNodeServiceClient(conn)
				_, err = client.CommitBlock(ctx, pb)
				conn.Close() // Đóng ngay sau call

				if err == nil {
					log.Printf("📨 Block committed broadcast to %s", peerAddr)
					return
				}

				lastErr = err
				log.Printf("❌ [Attempt %d] Failed to commit to %s: %v", attempt, peerAddr, err)
				time.Sleep(500 * time.Millisecond)
			}

			log.Printf("⛔ Exhausted retries for %s: %v", peerAddr, lastErr)
		}(addr)
	}
}

func SyncBlockFromLeader(startHeight int, leaderAddr string, db *storage.DB) error {
	conn, err := grpc.Dial(leaderAddr, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("failed to connect to leader: %v", err)
	}
	defer conn.Close()

	client := nodepb.NewNodeServiceClient(conn)

	res, err := client.GetBlockFromHeight(context.Background(), &nodepb.HeightRequest{FromHeight: int64(startHeight)})
	if err != nil {
		return fmt.Errorf("sync failed: %v", err)
	}

	for _, pb := range res.Blocks {
		block := blockchain.ProtoToBlock(pb)
		err := db.SaveBlock(block)
		if err != nil {
			return fmt.Errorf("failed to save synced block at height %d: %v", block.Height, err)
		}
		log.Printf("📥 Synced block at height %d", block.Height)
	}

	log.Printf("✅ Sync complete. Total blocks synced: %d", len(res.Blocks))
	return nil
}

func (s *NodeServer) processTransactionsAndUpdateStateOnFollower(transactions []*blockchain.Transaction, trie *mpt.MPT) ([]byte, error) {
	for _, tx := range transactions {
		senderAddr := tx.Sender
		receiverAddr := tx.Receiver

		// Lấy trạng thái tài khoản người gửi
		senderData, _ := s.StateTrie.Get(senderAddr)
		senderAccount := &blockchain.Account{Balance: 1000000} // Giả sử số dư ban đầu rất lớn để test
		if senderData != nil {
			senderAccount, _ = blockchain.DeserializeAccount(senderData)
		}

		// Kiểm tra số dư
		if senderAccount.Balance < tx.Amount {
			log.Printf("❌ Insufficient balance for sender %x", senderAddr)
			continue // Bỏ qua giao dịch không hợp lệ
		}

		// Lấy trạng thái tài khoản người nhận
		receiverData, _ := s.StateTrie.Get(receiverAddr)
		receiverAccount := &blockchain.Account{Balance: 0}
		if receiverData != nil {
			receiverAccount, _ = blockchain.DeserializeAccount(receiverData)
		}

		// Cập nhật số dư
		senderAccount.Balance -= tx.Amount
		receiverAccount.Balance += tx.Amount

		// Serialize và lưu lại vào Trie
		newSenderData, _ := senderAccount.Serialize()
		s.StateTrie.Insert(senderAddr, newSenderData)

		newReceiverData, _ := receiverAccount.Serialize()
		s.StateTrie.Insert(receiverAddr, newReceiverData)
	}

	// Trả về root hash mới của State Trie
	return s.StateTrie.RootHash(), nil
}
