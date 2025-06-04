package main

import (
	"blockchain-go/pkg/p2p"
	"os"
)

func main() {
	nodeID := os.Getenv("NODE_ID")
	port := os.Getenv("PORT")
	p2p.StartServer(port, nodeID)
}
