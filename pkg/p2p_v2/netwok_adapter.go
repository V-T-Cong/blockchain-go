package p2p_v2

import (
	"blockchain-go/pkg/blockchain"
	"blockchain-go/proto/nodepb"
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GrpcAdapter struct {
	leaderAddr string
	peerAddrs  []string
}

func NewGrpcAdapter(leaderAddr string, peerAddrs []string) *GrpcAdapter {
	return &GrpcAdapter{
		leaderAddr: leaderAddr,
		peerAddrs:  peerAddrs,
	}
}

// Send propose block to all followers
func (a *GrpcAdapter) BroadcastProposedBlock(block *blockchain.Block) {
	log.Printf("📤 Proposing block to %d followers...", len(a.peerAddrs))

	pb := blockchain.BlockToProto(block)

	for _, addr := range a.peerAddrs {
		peerAddr := addr // Tạo biến cục bộ cho goroutine
		go a.sendToPeer(peerAddr, func(client nodepb.NodeServiceClient) error {
			res, err := client.ProposeBlock(context.Background(), pb)
			if err != nil {
				return err
			}
			if !res.Success {
				return fmt.Errorf("rejected: %s", res.Message)
			}
			log.Printf("✅ Suggest block to %s success.", peerAddr)
			return nil
		})
	}
}

// Send vote for leader
func (a *GrpcAdapter) SendVoteToLeader(vote *nodepb.Vote) error {
	var err error
	a.sendToPeer(a.leaderAddr, func(client nodepb.NodeServiceClient) error {
		_, err = client.VoteBlock(context.Background(), vote)
		if err == nil {
			log.Printf("✅ Submitted vote for block %x to leader.", vote.BlockHash)
		}
		return err
	})
	return err
}

// Send message that block committed to all nodes
func (a *GrpcAdapter) BroadcastCommittedBlock(block *blockchain.Block) {
	log.Printf("📢 Notifying committed block %d to %d peers...", block.Height, len(a.peerAddrs))
	pb := blockchain.BlockToProto(block)

	for _, addr := range a.peerAddrs {
		peerAddr := addr // Tạo biến cục bộ
		go a.sendToPeer(peerAddr, func(client nodepb.NodeServiceClient) error {
			_, err := client.CommitBlock(context.Background(), pb)
			if err == nil {
				log.Printf("✅ Commit notification to %s success.", peerAddr)
			}
			return err
		})
	}
}

func (a *GrpcAdapter) sendToPeer(peerAddr string, rpcCall func(client nodepb.NodeServiceClient) error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, peerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())

	if err != nil {
		log.Printf("❌ can not connect to peer %s: %v", peerAddr, err)
		return
	}

	defer conn.Close()

	client := nodepb.NewNodeServiceClient(conn)
	if err := rpcCall(client); err != nil {
		log.Printf("❌ RPC error to peer %s: %v", peerAddr, err)
	}
}
