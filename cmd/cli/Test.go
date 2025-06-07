package main

import (
	"fmt"
	"log"
	"os"

	"blockchain-go/pkg/storage"
)

func main() {

	dbPath := "../../data/testdb"
	_ = os.MkdirAll(dbPath, os.ModePerm)
	db, err := storage.OpenDB(dbPath)
	if err != nil {
		log.Fatal("❌ Failed to open DB:", err)
	}
	defer db.Close()

	latestBlock, err := db.GetLatestBlock()

	fmt.Printf("Latest block:: %x", latestBlock)
}
