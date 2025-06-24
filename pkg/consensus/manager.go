package consensus

import (
	"blockchain-go/pkg/blockchain"
	"blockchain-go/pkg/state"
	"blockchain-go/pkg/storage"
	"blockchain-go/pkg/validation"
	"blockchain-go/proto/nodepb"
	"fmt"
	"log"
	"sync"
)

type Manager struct {
	NodeID         string
	DB             *storage.DB
	State          *state.State
	LatestBlock    *blockchain.Block
	PeerAddrs      []string
	TotalNodes     int
	LeaderAddr     string
	PendingBlocks  map[string]*blockchain.Block
	VoteCount      map[string]int
	BlockCommitted map[string]bool
	networker      Networker
	voteMutex      sync.Mutex
	netWorker      Networker
}

func NewManager(nodeID string, totalNodes int, db *storage.DB, s *state.State, latestBlock *blockchain.Block, networker Networker) *Manager {
	return &Manager{
		NodeID:         nodeID,
		TotalNodes:     totalNodes,
		DB:             db,
		State:          s,
		LatestBlock:    latestBlock,
		networker:      networker, // Gán networker
		PendingBlocks:  make(map[string]*blockchain.Block),
		VoteCount:      make(map[string]int),
		BlockCommitted: make(map[string]bool),
	}
}

func (m *Manager) HandleProposedBlock(block *blockchain.Block) error {
	log.Printf("📦 Validating proposed block at height %d", block.Height)

	if err := validation.ValidateBlock(block, m.State, m.LatestBlock); err != nil {
		return fmt.Errorf("validate block fail: %w", err)
	}

	log.Println("✅ Block pass all validation.")
	blockHash := string(block.CurrentBlockHash)
	m.PendingBlocks[blockHash] = block

	// Gửi phiếu bầu cho leader
	go func() {
		vote := &nodepb.Vote{
			VoterId:   m.NodeID,
			BlockHash: block.CurrentBlockHash,
			Approved:  true,
		}
		if err := m.networker.SendVoteToLeader(vote); err != nil {
			log.Printf("❌ can not send vote to leader : %v", err)
		} else {
			log.Printf("✅ Send vote to leader for block %x", block.CurrentBlockHash)
		}
	}()

	return nil
}

func (m *Manager) HandleVote(vote *nodepb.Vote) {
	log.Printf("🗳️  Receive vote from %s: approved=%v", vote.VoterId, vote.Approved)
	if !vote.Approved {
		return // Bỏ qua các vote không đồng ý
	}

	blockHashKey := string(vote.BlockHash)

	m.voteMutex.Lock()
	m.VoteCount[blockHashKey]++
	voteCount := m.VoteCount[blockHashKey]
	m.voteMutex.Unlock()

	needed := m.TotalNodes/2 + 1
	log.Printf("🗳️  Block %x có %d/%d vote.", vote.BlockHash, voteCount, needed)

	if voteCount >= needed && !m.BlockCommitted[blockHashKey] {
		log.Printf("🎉 Get enough votes for the block %x. Start commit...", vote.BlockHash)
		block := m.PendingBlocks[blockHashKey]
		if block == nil {
			log.Printf("⚠️ No pending block found %x to commit", vote.BlockHash)
			return
		}

		// Leader tự commit trước
		if err := m.CommitBlock(block); err != nil {
			log.Printf("🔥 Fatal error when Leader commit block: %v", err)
			return
		}

		// Thông báo cho các follower khác commit
		m.networker.BroadcastCommittedBlock(block)
	}
}

func (m *Manager) CreateAndProposeBlock(txs []*blockchain.Transaction) {
	prevHash := []byte{}
	height := 0
	if m.LatestBlock != nil {
		prevHash = m.LatestBlock.CurrentBlockHash
		height = int(m.LatestBlock.Height) + 1
	}

	block := blockchain.NewBlock(txs, prevHash, height)
	blockHashKey := string(block.CurrentBlockHash)
	log.Printf("📦 Leader: creating block at height %d with %d transactions", block.Height, len(txs))

	m.PendingBlocks[blockHashKey] = block

	m.voteMutex.Lock()
	m.VoteCount[blockHashKey] = 1 // Leader tự động vote cho chính mình
	m.voteMutex.Unlock()

	m.networker.BroadcastProposedBlock(block)
}

func (m *Manager) CommitBlock(block *blockchain.Block) error {
	blockHash := string(block.CurrentBlockHash)

	if m.BlockCommitted[blockHash] {
		log.Printf("⚠️ Block %d has been committed before", block.Height)
		return nil
	}

	// Xác thực lại lần cuối trước khi commit
	if err := validation.ValidateBlock(block, m.State, m.LatestBlock); err != nil {
		return fmt.Errorf("block validation failed on commit: %w", err)
	}

	if err := m.DB.SaveBlock(block); err != nil {
		return fmt.Errorf("save block fail: %w", err)
	}

	// Cập nhật State
	for _, tx := range block.Transactions {
		if err := m.State.ApplyTransaction(tx); err != nil {
			log.Printf("🔥 Fatal error: Cannot apply transaction in committed block: %v", err)
		}
	}
	log.Println("💰 Balance updated.")

	m.LatestBlock = block
	m.BlockCommitted[blockHash] = true
	log.Printf("✅ Block %d has been committed successfully", block.Height)

	return nil
}
