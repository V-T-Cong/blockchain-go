package p2p

import (
	"blockchain-go/pkg/blockchain"
	"blockchain-go/proto/nodepb"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"time"
)

type NodeClient struct {
	client nodepb.NodeServiceClient
}

func SendTransactionToPeer(addr string, tx *nodepb.Transaction) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(3*time.Second))
	if err != nil {
		log.Printf("❌ Connect failed: %v\n", err)
		return
	}
	defer conn.Close()
	client := nodepb.NewNodeServiceClient(conn)
	resp, err := client.SendTransaction(context.Background(), tx)
	if err != nil {
		log.Printf("❌ SendTransaction error: %v", err)
		return
	}
	log.Println("✅ Response:", resp.Message)
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

func ProposeBlockToFollowers(block *blockchain.Block, peerAddrs []string) {
	pb := blockchain.BlockToProto(block)

	for _, addr := range peerAddrs {
		log.Printf("📤 Proposing block to %s", addr)
		go func(peerAddr string) {
			conn, err := grpc.Dial(peerAddr, grpc.WithInsecure())
			if err != nil {
				log.Printf("❌ Cannot connect to %s: %v", peerAddr, err)
				return
			}
			defer conn.Close()

			client := nodepb.NewNodeServiceClient(conn)
			_, err = client.ProposeBlock(context.Background(), pb)
			if err != nil {
				log.Printf("❌ Failed to send block to %s: %v", peerAddr, err)
			} else {
				log.Printf("📤 Block proposed to %s", peerAddr)
			}
		}(addr)
	}
}

func BroadcastCommittedBlock(block *blockchain.Block, allNodeAddrs []string) {
	pb := blockchain.BlockToProto(block)

	for _, addr := range allNodeAddrs {
		go func(peerAddr string) {
			conn, err := grpc.Dial(peerAddr, grpc.WithInsecure())
			if err != nil {
				log.Printf("❌ Cannot connect to %s: %v", peerAddr, err)
				return
			}
			defer conn.Close()

			client := nodepb.NewNodeServiceClient(conn)
			_, err = client.CommitBlock(context.Background(), pb)
			if err != nil {
				log.Printf("❌ Failed to confirm block to %s: %v", peerAddr, err)
			} else {
				log.Printf("📨 Block confirmed to %s", peerAddr)
			}
		}(addr)
	}
}
