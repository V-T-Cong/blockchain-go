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

	// Save the block by its hash
	if err := d.db.Put(key, value, nil); err != nil {
		return fmt.Errorf("failed to save block: %w", err)
	}

	// Save the latest block hash
	if err := d.db.Put([]byte("latest"), key, nil); err != nil {
		return fmt.Errorf("failed to update latest block: %w", err)
	}

	// Save the height-to-hash index (e.g., "height-4" → blockHash)
	heightKey := []byte(fmt.Sprintf("height-%d", block.Height))
	if err := d.db.Put(heightKey, key, nil); err != nil {
		return fmt.Errorf("failed to save height index: %w", err)
	}

	return nil
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

func (d *DB) GetLatestBlock() (*blockchain.Block, error) {
	latestHash, err := d.db.Get([]byte("latest"), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest block hash: %w", err)
	}
	return d.GetBlock(latestHash)
}

func (d *DB) GetAllBlocks() ([]*blockchain.Block, error) {
	var blocks []*blockchain.Block
	iter := d.db.NewIterator(nil, nil)
	for iter.Next() {
		key := iter.Key()
		if string(key) == "latest" {
			continue
		}
		val := iter.Value()
		var block blockchain.Block
		if err := json.Unmarshal(val, &block); err != nil {
			return nil, err
		}
		blocks = append(blocks, &block)
	}
	iter.Release()
	return blocks, iter.Error()
}

func (d *DB) GetBlockByHeight(height int) (*blockchain.Block, error) {
	heightKey := []byte(fmt.Sprintf("height-%d", height))
	hash, err := d.db.Get(heightKey, nil)
	if err != nil {
		return nil, fmt.Errorf("height index not found: %w", err)
	}
	return d.GetBlock(hash)
}

// Get retrieves a value by key.
func (d *DB) Get(key []byte) ([]byte, error) {
	return d.db.Get(key, nil)
}

// Put saves a key-value pair.
func (d *DB) Put(key, value []byte, options ...interface{}) error {
	return d.db.Put(key, value, nil)
}

// Close closes the DB
func (d *DB) Close() error {
	return d.db.Close()
}
