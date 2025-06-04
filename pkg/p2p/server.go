package p2p

import (
	"context"
	"fmt"
	"log"
	"net"

	"blockchain-go/blockchain/pkg/p2p/nodepb"
	"google.golang.org/grpc"
)

type NodeServer struct {
	nodepb.UnimplementedNodeServiceServer
	NodeID string
}

func (s *NodeServer) SendTransaction(ctx context.Context, tx *nodepb.Transaction) (*nodepb.Ack, error) {
	log.Printf("Received transaction from: %x → %x\n", tx.Sender, tx.Receiver)
	return &nodepb.Ack{Message: "Transaction received"}, nil
}

func (s *NodeServer) ProposeBlock(ctx context.Context, b *nodepb.Block) (*nodepb.Ack, error) {
	log.Printf("Received block proposal with %d transactions\n", len(b.Transactions))
	return &nodepb.Ack{Message: "Block received"}, nil
}

func (s *NodeServer) Vote(ctx context.Context, v *nodepb.Vote) (*nodepb.Ack, error) {
	log.Printf("Received vote from %s - accepted: %v\n", v.NodeId, v.Accepted)
	return &nodepb.Ack{Message: "Vote received"}, nil
}

func (s *NodeServer) GetBlock(ctx context.Context, req *nodepb.BlockRequest) (*nodepb.BlockResponse, error) {
	log.Printf("GetBlock request at height: %d\n", req.Height)
	return &nodepb.BlockResponse{}, nil
}

func StartServer(port string, nodeID string) {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("❌ Failed to listen: %v", err)
	}
	server := grpc.NewServer()
	nodepb.RegisterNodeServiceServer(server, &NodeServer{NodeID: nodeID})
	fmt.Printf("🚀 Node %s listening on port %s\n", nodeID, port)
	if err := server.Serve(lis); err != nil {
		log.Fatalf("❌ Failed to serve: %v", err)
	}
}
