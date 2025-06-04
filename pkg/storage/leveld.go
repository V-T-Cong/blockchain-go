package storage

import (
	"blockchain-go/pkg/blockchain"
	"encoding/json"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
)

type DB struct {
	db *leveldb.DB
}

// OpenDB opens or creates the database at a given path.
func OpenDB(path string) (*DB, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	return &DB{db: db}, nil
}

func (d *DB) SaveBlock(block *blockchain.Block) error {
	key := block.CurrentBlockHash
	value, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("failed to marshal block: %w", err)
	}
	return d.db.Put(key, value, nil)
}

// GetBlock retrieves a block by hash.
func (d *DB) GetBlock(hash []byte) (*blockchain.Block, error) {
	value, err := d.db.Get(hash, nil)
	if err != nil {
		return nil, fmt.Errorf("block not found: %w", err)
	}
	var block blockchain.Block
	if err := json.Unmarshal(value, &block); err != nil {
		return nil, fmt.Errorf("failed to decode block: %w", err)
	}
	return &block, nil
}

// Close closes the DB
func (d *DB) Close() error {
	return d.db.Close()
}
