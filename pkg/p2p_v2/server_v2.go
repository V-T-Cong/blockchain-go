// pkg/p2p_v2/server_v2.go
package p2p_v2

import (
	"blockchain-go/pkg/blockchain"
	"blockchain-go/pkg/consensus"
	"blockchain-go/pkg/cryptohelper"
	"blockchain-go/pkg/state"
	"blockchain-go/proto/nodepb"
	"context"
	"encoding/hex"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type NodeServer struct {
	nodepb.UnimplementedNodeServiceServer

	NodeID   string
	IsLeader bool

	// Fields solve transaction
	PendingTxs  []*blockchain.Transaction
	txMutex     sync.Mutex
	isCreating  bool
	createMutex sync.Mutex

	// modules handle logic
	Consensus *consensus.Manager
	State     *state.State
}

// SendTransaction nhận một giao dịch mới
func (s *NodeServer) SendTransaction(ctx context.Context, txProto *nodepb.Transaction) (*nodepb.Status, error) {
	txInternal := blockchain.ProtoToBlock(&nodepb.Block{Transactions: []*nodepb.Transaction{txProto}}).Transactions[0]

	// Xác thực chữ ký cơ bản
	pubKey, err := cryptohelper.BytesToPublicKey(txInternal.PublicKey)
	if err != nil {
		return &nodepb.Status{Message: "Public key Invalid", Success: false}, nil
	}
	if !blockchain.VerifyTransaction(txInternal, pubKey) {
		return &nodepb.Status{Message: "Chữ ký không hợp lệ", Success: false}, nil
	}

	// Nếu là Leader, kiểm tra số dư ngay lập tức
	if s.IsLeader {
		senderKey := hex.EncodeToString(txInternal.Sender)
		balance, err := s.State.GetBalance(senderKey)
		if err != nil {
			return &nodepb.Status{Message: "Error when checked balance", Success: false}, nil
		}
		if balance < txInternal.Amount {
			return &nodepb.Status{Message: "balance not enough", Success: false}, nil
		}
	}

	// Add to queue and trigger block creation if needed
	s.addTxToPending(txInternal)

	return &nodepb.Status{Message: "Received transaction", Success: true}, nil
}

func (s *NodeServer) addTxToPending(tx *blockchain.Transaction) {
	s.txMutex.Lock()
	s.PendingTxs = append(s.PendingTxs, tx)
	txCount := len(s.PendingTxs)
	s.txMutex.Unlock()

	if !s.IsLeader {
		return
	}

	s.createMutex.Lock()
	if s.isCreating {
		s.createMutex.Unlock()
		return
	}
	// Nếu đủ 10 tx hoặc đã đến lúc, tạo block
	if txCount >= 10 {
		s.isCreating = true
		s.createMutex.Unlock()
		go s.triggerCreateBlock()
	} else {
		s.createMutex.Unlock()
		// Tạo một timer, nếu sau 5s chưa có block mới thì sẽ tạo
		time.AfterFunc(5*time.Second, func() {
			s.createMutex.Lock()
			if !s.isCreating {
				s.isCreating = true
				s.createMutex.Unlock()
				go s.triggerCreateBlock()
			} else {
				s.createMutex.Unlock()
			}
		})
	}
}

func (s *NodeServer) triggerCreateBlock() {
	s.txMutex.Lock()
	if len(s.PendingTxs) == 0 {
		s.txMutex.Unlock()
		s.createMutex.Lock()
		s.isCreating = false
		s.createMutex.Unlock()
		return
	}

	// Lấy tối đa 10 giao dịch
	var txsToProcess []*blockchain.Transaction
	if len(s.PendingTxs) > 10 {
		txsToProcess = s.PendingTxs[:10]
		s.PendingTxs = s.PendingTxs[10:]
	} else {
		txsToProcess = s.PendingTxs
		s.PendingTxs = []*blockchain.Transaction{}
	}
	s.txMutex.Unlock()

	// Gọi Consensus Manager để xử lý
	s.Consensus.CreateAndProposeBlock(txsToProcess)

	// Reset cờ
	time.Sleep(2 * time.Second) // Chờ một chút trước khi cho phép tạo block mới
	s.createMutex.Lock()
	s.isCreating = false
	s.createMutex.Unlock()
}

// ProposeBlock là RPC handler cho follower.
func (s *NodeServer) ProposeBlock(ctx context.Context, pb *nodepb.Block) (*nodepb.Status, error) {
	if s.IsLeader {
		return &nodepb.Status{Message: "Leader does not accept the proposal", Success: false}, nil
	}

	block := blockchain.ProtoToBlock(pb)
	err := s.Consensus.HandleProposedBlock(block) // Ủy quyền cho Consensus Manager
	if err != nil {
		log.Printf("❌ Block was rejected: %v", err)
		return &nodepb.Status{Message: err.Error(), Success: false}, nil
	}

	return &nodepb.Status{Message: "Block has been verified, sending vote.", Success: true}, nil
}

// VoteBlock là RPC handler cho leader.
func (s *NodeServer) VoteBlock(ctx context.Context, vote *nodepb.Vote) (*nodepb.Status, error) {
	if !s.IsLeader {
		return &nodepb.Status{Message: "Only leader receive vote", Success: false}, nil
	}

	go s.Consensus.HandleVote(vote) // Xử lý bất đồng bộ

	return &nodepb.Status{Message: "Received vote", Success: true}, nil
}

// CommitBlock là RPC handler cho follower để commit block đã được đồng thuận.
func (s *NodeServer) CommitBlock(ctx context.Context, pb *nodepb.Block) (*nodepb.Status, error) {
	if s.IsLeader {
		return &nodepb.Status{Message: "Leader commits by himself, no need for this RPC", Success: true}, nil
	}

	block := blockchain.ProtoToBlock(pb)
	if err := s.Consensus.CommitBlock(block); err != nil {
		log.Printf("❌ Follower commit block fail: %v", err)
		return &nodepb.Status{Message: err.Error(), Success: false}, nil
	}

	return &nodepb.Status{Message: "Block has been committed", Success: true}, nil
}

// GetBlockFromHeight trả về danh sách các block từ một height nhất định.
func (s *NodeServer) GetBlockFromHeight(ctx context.Context, req *nodepb.HeightRequest) (*nodepb.BlockList, error) {
	start := int(req.FromHeight)
	var blocks []*nodepb.Block

	for h := start; ; h++ {
		block, err := s.Consensus.DB.GetBlockByHeight(h)
		if err != nil {
			break // Hết block
		}
		blocks = append(blocks, blockchain.BlockToProto(block))
	}

	return &nodepb.BlockList{Blocks: blocks}, nil
}

// GetBalance return balance from the address
func (s *NodeServer) GetBalance(ctx context.Context, req *nodepb.GetBalanceRequest) (*nodepb.GetBalanceResponse, error) {
	log.Printf("🔍 Received GetBalance request for address: %s", req.Address)
	balance, err := s.State.GetBalance(req.Address)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Can not get balance: %v", err)
	}

	return &nodepb.GetBalanceResponse{
		Address: req.Address,
		Balance: balance,
	}, nil
}
