package p2p_v2

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"blockchain-go/pkg/blockchain"
	"blockchain-go/pkg/cryptohelper"
	"blockchain-go/pkg/mpt"
	"blockchain-go/pkg/storage"
	"blockchain-go/proto/nodepb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	// "golang.org/x/tools/go/analysis/passes/defers"
	// "google.golang.org/grpc/codes"
	// "google.golang.org/grpc/status"
)

type NodeServer struct {
	nodepb.UnimplementedNodeServiceServer

	NodeID      string
	IsLeader    bool
	LeaderAddr  string
	LeaderID    string
	DB          *storage.DB
	LatestBlock *blockchain.Block
	StateTrie   *mpt.MPT // state trie for account balances

	PendingTxs []*blockchain.Transaction

	// Consensus-related
	PendingBlocks  map[string]*blockchain.Block // blockHash → block
	VoteCount      map[string]int               // blockHash → vote count
	BlockCommitted map[string]bool              // blockHash → committed status
	VoteMutex      sync.Mutex                   // protect voteCount

	PeerAddrs  []string
	TotalNodes int

	// handler error
	IsCommitting bool
	BlockMutex   sync.Mutex
	VotedBlocks  map[string]bool
	VoteReceived map[string]map[string]bool
}

func (s *NodeServer) GetBalance(ctx context.Context, req *nodepb.GetBalanceRequest) (*nodepb.GetBalanceResponse, error) {
	address := req.Address
	log.Printf("🔍 Received GetBalance request for address: %x", address)

	accountData, ok := s.StateTrie.Get(address)
	if !ok || accountData == nil {
		// Trả về số dư 0 nếu tài khoản chưa có trong state
		return &nodepb.GetBalanceResponse{Balance: 0, Nonce: 0}, nil
	}

	account, err := blockchain.DeserializeAccount(accountData)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to deserialize account data")
	}

	return &nodepb.GetBalanceResponse{
		Balance: account.Balance,
		Nonce:   account.Nonce,
	}, nil
}

func (s *NodeServer) SendTransaction(ctx context.Context, tx *nodepb.Transaction) (*nodepb.Status, error) {
	// fmt.Println("📩 Received transaction")

	txInternal := &blockchain.Transaction{
		Sender:    tx.Sender,
		Receiver:  tx.Receiver,
		Amount:    tx.Amount,
		Timestamp: tx.Timestamp,
		Signature: tx.Signature,
		PublicKey: tx.PublicKey,
	}

	// Verify transaction and public key
	pubKey, err := cryptohelper.BytesToPublicKey(tx.PublicKey)
	if err != nil {
		fmt.Printf("Failed to parse public key: %v", err)
		return &nodepb.Status{Message: "Invalid Signature", Success: false}, nil
	}

	if !blockchain.VerifyTransaction(txInternal, pubKey) {
		log.Println("❌ Invalid signature")
		return &nodepb.Status{Message: "Invalid signature", Success: false}, nil
	}

	s.PendingTxs = append(s.PendingTxs, txInternal)
	// log.Printf("✅ Tx added. Total pending: %d\n", len(s.PendingTxs))

	// Create block after 5 second or colect enought 10 transactions
	if s.NodeID == "node1" {
		if len(s.PendingTxs) >= 10 {
			go s.createBlockFromPending()
		} else {
			go func() {
				time.Sleep(5 * time.Second)

				s.BlockMutex.Lock()
				defer s.BlockMutex.Unlock()

				if len(s.PendingTxs) > 0 && !s.IsCommitting {
					s.IsCommitting = true
					go s.createBlockFromPending()
				}
			}()
		}
	}

	return &nodepb.Status{Message: "Transaction received", Success: true}, nil
}

func (s *NodeServer) createBlockFromPending() {
	s.BlockMutex.Lock()
	defer s.BlockMutex.Unlock()

	if len(s.PendingTxs) == 0 {
		log.Println("⚠️ No transactions to create a block")
		s.IsCommitting = false
		return
	}

	prevHash := []byte{}
	height := 0
	if s.LatestBlock != nil {
		prevHash = s.LatestBlock.CurrentBlockHash
		height = int(s.LatestBlock.Height) + 1
	}

	// Take up to 10 txs for the block
	txs := s.PendingTxs
	if len(txs) > 10 {
		txs = s.PendingTxs[:10]
		s.PendingTxs = s.PendingTxs[10:]
	} else {
		s.PendingTxs = nil
	}

	tempStateTrie := s.StateTrie.Clone()
	tempAccounts := make(map[string]*blockchain.Account)

	for _, tx := range txs {
		senderAddrHex := hex.EncodeToString(tx.Sender)
		receiverAddrHex := hex.EncodeToString(tx.Receiver)
		var senderAccount, receiverAccount *blockchain.Account
		if acc, ok := tempAccounts[senderAddrHex]; ok {
			senderAccount = acc
		} else {
			senderData, _ := tempStateTrie.Get(tx.Sender)
			if senderData == nil {
				senderAccount = &blockchain.Account{Balance: 1000000}
			} else {
				senderAccount, _ = blockchain.DeserializeAccount(senderData)
			}
		}
		if acc, ok := tempAccounts[receiverAddrHex]; ok {
			receiverAccount = acc
		} else {
			receiverData, _ := tempStateTrie.Get(tx.Receiver)
			if receiverData == nil {
				receiverAccount = &blockchain.Account{Balance: 0}
			} else {
				receiverAccount, _ = blockchain.DeserializeAccount(receiverData)
			}
		}
		if senderAccount.Balance >= tx.Amount {
			senderAccount.Balance -= tx.Amount
			receiverAccount.Balance += tx.Amount
		} else {
			continue
		}
		tempAccounts[senderAddrHex] = senderAccount
		tempAccounts[receiverAddrHex] = receiverAccount
	}
	var sortedAddrs []string
	for addrHex := range tempAccounts {
		sortedAddrs = append(sortedAddrs, addrHex)
	}
	sort.Strings(sortedAddrs)
	for _, addrHex := range sortedAddrs {
		account := tempAccounts[addrHex]
		addrBytes, _ := hex.DecodeString(addrHex)
		accountData, _ := account.Serialize()
		tempStateTrie.Insert(addrBytes, accountData)
	}

	stateRoot := tempStateTrie.RootHash()

	block := blockchain.NewBlock(txs, prevHash, height)
	block.StateRoot = stateRoot

	block.CurrentBlockHash = block.Hash()

	blockHash := string(block.CurrentBlockHash)

	log.Printf("📦 Leader: Creating block at height %d with StateRoot %x", block.Height, stateRoot)

	s.PendingBlocks[blockHash] = block

	// Propose to followers
	go ProposeBlockToFollowers(block, s.PeerAddrs)

	// Reset commit flag after delay
	go func() {
		time.Sleep(2 * time.Second)
		s.BlockMutex.Lock()
		s.IsCommitting = false
		s.BlockMutex.Unlock()
	}()
}

func (s *NodeServer) ProposeBlock(ctx context.Context, pb *nodepb.Block) (*nodepb.Status, error) {
	log.Println("🔔 ProposeBlock CALLED on follower!")
	log.Println("📦 Received proposed block")

	block := blockchain.ProtoToBlock(pb)

	// verify transaction signatures
	for _, tx := range block.Transactions {
		pubKey, err := cryptohelper.BytesToPublicKey(tx.PublicKey)
		if err != nil {
			log.Println("❌ Invalid public key")
			return &nodepb.Status{Message: "Invalid public key", Success: false}, nil
		}
		if !blockchain.VerifyTransaction(tx, pubKey) {
			log.Println("❌ Invalid signature in tx")
			return &nodepb.Status{Message: "Invalid signature in tx", Success: false}, nil
		}
	}
	// log.Println("✅ Signatures OK")

	// merkle validity
	var txHashes [][]byte
	for _, tx := range block.Transactions {
		txHashes = append(txHashes, tx.Hash())
	}

	_, computedRoot := mpt.BuildMPTFromTxHashes(txHashes)

	if !bytes.Equal(computedRoot, block.MerkleRoot) {
		log.Printf("❌ Merkle root mismatch\nExpected: %x\nGot: %x", block.MerkleRoot, computedRoot)
		return &nodepb.Status{Message: "Merkle root mismatch", Success: false}, nil
	}
	// log.Println("✅ Merkle Root OK")

	// verify previousblock
	if s.LatestBlock != nil {
		if !bytes.Equal(block.PreviousBlockHash, s.LatestBlock.CurrentBlockHash) {
			log.Println("❌ Previous block hash mismatch")
			return &nodepb.Status{Message: "PreviousBlockHash mismatch", Success: false}, nil
		}
	}
	// log.Println("✅ PrevBlockHash OK")

	// validate state root
	if !s.validateStateRoot(block) {
		// Hàm validateStateRoot đã log lỗi chi tiết
		return &nodepb.Status{Message: "Invalid State Root", Success: false}, nil
	}
	// log.Println("✅ StateRoot OK")

	blockHash := string(block.CurrentBlockHash)
	s.PendingBlocks[blockHash] = block

	log.Println("✅ Block passed all verification checks")

	go func() {
		vote := &nodepb.Vote{
			VoterId:   s.NodeID,
			BlockHash: block.CurrentBlockHash,
			Approved:  true,
		}

		err := SendVoteToLeader(vote, s.LeaderAddr)
		if err != nil {
			log.Printf("❌ Failed to send vote to leader: %v", err)
		} else {
			log.Printf("✅ Vote sent to leader for block %x", block.CurrentBlockHash)
		}
	}()

	return &nodepb.Status{Message: "Block verified", Success: true}, nil
}

func (s *NodeServer) VoteBlock(ctx context.Context, vote *nodepb.Vote) (*nodepb.Status, error) {
	fmt.Printf("🗳️ Received vote from %s: approved=%v\n", vote.VoterId, vote.Approved)

	blockHashKey := string(vote.BlockHash)

	// Leader vote skip
	if vote.VoterId == s.NodeID && s.IsLeader {
		return &nodepb.Status{Message: "Vote ignored", Success: true}, nil
	}

	block := s.PendingBlocks[blockHashKey]
	if block == nil {
		return &nodepb.Status{Message: "Block not found", Success: false}, nil
	}

	// Handle case when only one node exists
	if s.TotalNodes == 1 {
		block := s.PendingBlocks[blockHashKey]
		if block == nil {
			return &nodepb.Status{Message: "Block not found", Success: false}, nil
		}
		if !s.BlockCommitted[blockHashKey] {
			status, err := s.CommitBlock(ctx, blockchain.BlockToProto(block))
			if err != nil || !status.Success {
				return &nodepb.Status{Message: "Commit failed", Success: false}, nil
			}
			s.BlockCommitted[blockHashKey] = true
			fmt.Printf("✅ Block committed (single-node)! Height: %d\n", block.Height)
		}
		return &nodepb.Status{Message: "Block committed (single-node)", Success: true}, nil
	}

	if vote.Approved {
		s.VoteMutex.Lock()
		s.VoteCount[blockHashKey]++
		voteCount := s.VoteCount[blockHashKey]
		s.VoteMutex.Unlock()

		needed := s.TotalNodes/2 + 1

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
	blockHash := string(block.CurrentBlockHash)

	// Tránh commit trùng
	if s.BlockCommitted[blockHash] {
		log.Printf("⚠️ Block %d already committed", block.Height)
		return &nodepb.Status{Message: "Already committed", Success: true}, nil
	}

	// Validate block again (optional nhưng khuyên dùng)
	if !blockchain.ValidateBlock(block, s.LatestBlock) {
		log.Printf("❌ Invalid block during commit: height %d", block.Height)
		return &nodepb.Status{Message: "Invalid block on commit", Success: false}, nil
	}

	_, err := s.processAndUpdateState(block.Transactions)
	if err != nil {
		log.Printf("❌ Failed to update state on commit for block %d: %v", block.Height, err)
		return &nodepb.Status{Message: "Failed to update state on commit", Success: false}, nil
	}
	log.Printf("⛓️  State Trie updated for committed block %d", block.Height)

	// Save block to LevelDB
	if err := s.DB.SaveBlock(block); err != nil {
		log.Printf("❌ Failed to save block: %v", err)
		return &nodepb.Status{Message: "Failed to save block", Success: false}, nil
	}

	// Update state
	s.LatestBlock = block
	s.BlockCommitted[blockHash] = true
	log.Printf("✅ Block %d committed successfully", block.Height)

	return &nodepb.Status{Message: "Block committed", Success: true}, nil
}

func (s *NodeServer) GetBlockFromHeight(ctx context.Context, req *nodepb.HeightRequest) (*nodepb.BlockList, error) {
	start := int(req.FromHeight)
	var blocks []*nodepb.Block

	for h := start; ; h++ {
		block, err := s.DB.GetBlockByHeight(h)
		if err != nil {
			break // hết block rồi
		}
		blocks = append(blocks, blockchain.BlockToProto(block))
	}

	return &nodepb.BlockList{Blocks: blocks}, nil
}

func (s *NodeServer) processAndUpdateState(transactions []*blockchain.Transaction) ([]byte, error) {
	// Dùng cache để xử lý các giao dịch liên quan đến cùng một tài khoản trong 1 block
	tempAccounts := make(map[string]*blockchain.Account)

	for _, tx := range transactions {
		senderAddrHex := hex.EncodeToString(tx.Sender)
		receiverAddrHex := hex.EncodeToString(tx.Receiver)

		var senderAccount, receiverAccount *blockchain.Account

		// Lấy trạng thái người gửi (ưu tiên cache)
		if acc, ok := tempAccounts[senderAddrHex]; ok {
			senderAccount = acc
		} else {
			senderData, _ := s.StateTrie.Get(tx.Sender)
			if senderData == nil {
				senderAccount = &blockchain.Account{Balance: 1000000} // Số dư ban đầu để test
			} else {
				senderAccount, _ = blockchain.DeserializeAccount(senderData)
			}
		}

		// Lấy trạng thái người nhận (ưu tiên cache)
		if acc, ok := tempAccounts[receiverAddrHex]; ok {
			receiverAccount = acc
		} else {
			receiverData, _ := s.StateTrie.Get(tx.Receiver)
			if receiverData == nil {
				receiverAccount = &blockchain.Account{Balance: 0}
			} else {
				receiverAccount, _ = blockchain.DeserializeAccount(receiverData)
			}
		}

		// Cập nhật số dư
		if senderAccount.Balance >= tx.Amount {
			senderAccount.Balance -= tx.Amount
			receiverAccount.Balance += tx.Amount
		} else {
			log.Printf("Sender %s has insufficient funds for tx, skipping", senderAddrHex)
			continue // Bỏ qua giao dịch không hợp lệ
		}

		// Cập nhật lại vào cache
		tempAccounts[senderAddrHex] = senderAccount
		tempAccounts[receiverAddrHex] = receiverAccount
	}

	// === PHẦN QUAN TRỌNG NHẤT: ĐẢM BẢO TÍNH TẤT ĐỊNH ===
	// 1. Lấy tất cả các địa chỉ (keys) từ map
	var sortedAddrs []string
	for addrHex := range tempAccounts {
		sortedAddrs = append(sortedAddrs, addrHex)
	}

	// 2. Sắp xếp các địa chỉ để có thứ tự cố định
	sort.Strings(sortedAddrs)

	// 3. Cập nhật vào Trie THEO THỨ TỰ ĐÃ SẮP XẾP
	for _, addrHex := range sortedAddrs {
		account := tempAccounts[addrHex]
		addrBytes, _ := hex.DecodeString(addrHex)
		accountData, _ := account.Serialize()
		s.StateTrie.Insert(addrBytes, accountData)
	}

	return s.StateTrie.RootHash(), nil
}

func (s *NodeServer) validateStateRoot(block *blockchain.Block) bool {
	// Dùng bản sao của Trie để xác thực an toàn
	tempStateTrie := s.StateTrie.Clone()
	tempAccounts := make(map[string]*blockchain.Account)

	for _, tx := range block.Transactions {
		senderAddrHex := hex.EncodeToString(tx.Sender)
		receiverAddrHex := hex.EncodeToString(tx.Receiver)
		var senderAccount, receiverAccount *blockchain.Account

		if acc, ok := tempAccounts[senderAddrHex]; ok {
			senderAccount = acc
		} else {
			senderData, _ := tempStateTrie.Get(tx.Sender)
			if senderData == nil {
				senderAccount = &blockchain.Account{Balance: 1000000}
			} else {
				senderAccount, _ = blockchain.DeserializeAccount(senderData)
			}
		}

		if acc, ok := tempAccounts[receiverAddrHex]; ok {
			receiverAccount = acc
		} else {
			receiverData, _ := tempStateTrie.Get(tx.Receiver)
			if receiverData == nil {
				receiverAccount = &blockchain.Account{Balance: 0}
			} else {
				receiverAccount, _ = blockchain.DeserializeAccount(receiverData)
			}
		}

		if senderAccount.Balance < tx.Amount {
			log.Printf("❌ [Validation] Insufficient balance for sender %s", senderAddrHex)
			return false
		}
		senderAccount.Balance -= tx.Amount
		receiverAccount.Balance += tx.Amount
		tempAccounts[senderAddrHex] = senderAccount
		tempAccounts[receiverAddrHex] = receiverAccount
	}

	// === LOGIC TƯƠNG TỰ: SẮP XẾP ĐỂ ĐẢM BẢO TÍNH TẤT ĐỊNH ===
	var sortedAddrs []string
	for addrHex := range tempAccounts {
		sortedAddrs = append(sortedAddrs, addrHex)
	}
	sort.Strings(sortedAddrs)

	for _, addrHex := range sortedAddrs {
		account := tempAccounts[addrHex]
		addrBytes, _ := hex.DecodeString(addrHex)
		accountData, _ := account.Serialize()
		tempStateTrie.Insert(addrBytes, accountData)
	}

	computedStateRoot := tempStateTrie.RootHash()

	if !bytes.Equal(computedStateRoot, block.StateRoot) {
		log.Printf("❌ State root mismatch!\n  Expected: %x\n  Got:      %x", block.StateRoot, computedStateRoot)
		return false
	}
	return true
}
