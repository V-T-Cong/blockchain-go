package p2p

import (
	"context"
	"fmt"
	"log"
	"sync"

	"blockchain-go/pkg/blockchain"
	"blockchain-go/pkg/cryptohelper"
	"blockchain-go/pkg/storage"
	"blockchain-go/proto/nodepb"
)

type NodeServer struct {
	nodepb.UnimplementedNodeServiceServer

	NodeID      string
	LeaderAddr  string
	DB          *storage.DB
	LatestBlock *blockchain.Block

	PendingTxs []*blockchain.Transaction

	// Consensus-related
	PendingBlocks  map[string]*blockchain.Block // blockHash → block
	VoteCount      map[string]int               // blockHash → vote count
	BlockCommitted map[string]bool              // blockHash → committed status
	VoteMutex      sync.Mutex                   // protect voteCount

	PeerAddrs  []string
	TotalNodes int
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
	pubKey, err := cryptohelper.BytesToPublicKey(tx.PublicKey)
	if err != nil {
		log.Printf("Failed to parse public key: %v", err)
		return &nodepb.Status{Success: false, Message: "Invalid public key"}, nil
	}

	if !blockchain.VerifyTransaction(txInternal, pubKey) {
		log.Println("❌ Invalid signature")
		return &nodepb.Status{Message: "Invalid signature", Success: false}, nil
	}

	s.PendingTxs = append(s.PendingTxs, txInternal)
	log.Printf("✅ Tx added. Total pending: %d\n", len(s.PendingTxs))

	if s.NodeID == "node1" && len(s.PendingTxs) >= 1 {
		log.Println("📦 Leader: creating block from pending transactions...")

		prevHash := []byte{}
		height := 0
		if s.LatestBlock != nil {
			prevHash = s.LatestBlock.CurrentBlockHash
			height = int(s.LatestBlock.Height) + 1
		}

		block := blockchain.NewBlock(s.PendingTxs, prevHash, height)
		s.PendingTxs = nil // Clear tx pool

		// Lưu lại block để chờ vote
		s.PendingBlocks[string(block.CurrentBlockHash)] = block

		if len(s.PeerAddrs) > 0 {
			log.Printf("📤 Proposing block to %d followers...\n", len(s.PeerAddrs))
			go ProposeBlockToFollowers(block, s.PeerAddrs) // ✅ CHỈ GỌI 1 LẦN (async)
		}
	}

	return &nodepb.Status{Message: "Transaction received", Success: true}, nil
}

func (s *NodeServer) ProposeBlock(ctx context.Context, pb *nodepb.Block) (*nodepb.Status, error) {
	fmt.Println("📦 Received proposed block")

	// 1. Convert proto block -> internal struct
	block := blockchain.ProtoToBlock(pb)

	// 2. Validate block
	if !blockchain.ValidateBlock(block, s.LatestBlock) {
		return &nodepb.Status{Message: "Invalid block", Success: false}, nil
	}

	// 3. Send vote back to leader
	vote := &nodepb.Vote{
		VoterId:     s.NodeID,
		BlockHash:   block.CurrentBlockHash,
		BlockHeight: block.Height,
		Approved:    true,
	}

	err := SendVoteToLeader(vote, s.LeaderAddr)
	if err != nil {
		return &nodepb.Status{Message: "Failed to send vote", Success: false}, nil
	}

	return &nodepb.Status{Message: "Block accepted, vote sent", Success: true}, nil

}

func (s *NodeServer) VoteBlock(ctx context.Context, vote *nodepb.Vote) (*nodepb.Status, error) {
	fmt.Printf("🗳️ Received vote from %s: approved=%v\n", vote.VoterId, vote.Approved)

	blockHashKey := string(vote.BlockHash)

	if vote.Approved {
		s.VoteMutex.Lock()
		s.VoteCount[blockHashKey]++
		voteCount := s.VoteCount[blockHashKey]
		s.VoteMutex.Unlock()

		needed := s.TotalNodes/2 + 1

		// Majority vote = 2 in 3 nodes
		if voteCount >= needed && !s.BlockCommitted[blockHashKey] {
			block := s.PendingBlocks[blockHashKey]
			if block == nil {
				return &nodepb.Status{Message: "Block not found", Success: false}, nil
			}

			status, err := s.CommitBlock(ctx, blockchain.BlockToProto(block))
			if err != nil || !status.Success {
				return &nodepb.Status{Message: "Commit failed", Success: false}, nil
			}
			s.BlockCommitted[blockHashKey] = true
			fmt.Printf("✅ Block committed! Height: %d\n", block.Height)
		}
	}

	return &nodepb.Status{Message: "Vote received", Success: true}, nil
}

func (s *NodeServer) CommitBlock(ctx context.Context, pb *nodepb.Block) (*nodepb.Status, error) {
	block := blockchain.ProtoToBlock(pb)

	// Validate again if muốn an toàn
	if !blockchain.ValidateBlock(block, s.LatestBlock) {
		return &nodepb.Status{Message: "Invalid block on commit", Success: false}, nil
	}

	// Save to DB
	if err := s.DB.SaveBlock(block); err != nil {
		return &nodepb.Status{Message: "Failed to save block", Success: false}, nil
	}

	s.LatestBlock = block
	log.Printf("📦 Block %d committed by broadcast", block.Height)

	return &nodepb.Status{Message: "Block committed", Success: true}, nil
}
