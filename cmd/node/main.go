package main

import (
	"log"
	"net"

	"blockchain-go/pkg/p2p"
	"blockchain-go/proto/nodepb"
	"google.golang.org/grpc"
)

func main() {
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("❌ Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	server := &p2p.NodeServer{}
	nodepb.RegisterNodeServiceServer(grpcServer, server)

	log.Println("🚀 Node gRPC server started on :50051")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("❌ Failed to serve: %v", err)
	}
}
