package consensus

import (
	"blockchain-go/pkg/blockchain"
	"blockchain-go/proto/nodepb"
)

type Networker interface {
	BroadcastProposedBlock(block *blockchain.Block)
	BroadcastCommittedBlock(block *blockchain.Block)
	SendVoteToLeader(vote *nodepb.Vote) error
}
