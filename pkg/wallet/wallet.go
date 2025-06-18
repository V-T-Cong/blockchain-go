package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
)

type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
	Address    string
}

type WalletJSON struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
	Address    string `json:"address"`
}

// Generate a new ECDSA key pair
func CreateWallet() (*Wallet, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	pub := &priv.PublicKey
	address := PublicKeyToAddress(pub)
	return &Wallet{
		PrivateKey: priv,
		PublicKey:  pub,
		Address:    address,
	}, nil
}

// Convert public key to address (20 bytes from SHA256 hash)
func PublicKeyToAddress(pub *ecdsa.PublicKey) string {
	pubBytes := append(pub.X.Bytes(), pub.Y.Bytes()...)
	hash := sha256.Sum256(pubBytes)
	return hex.EncodeToString(hash[len(hash)-20:])
}

// Save wallet to JSON file
func (w *Wallet) SaveToFile(filePath string) error {

	privBytes := w.PrivateKey.D.Bytes()

	pubKeyBytes := append(w.PublicKey.X.Bytes(), w.PublicKey.Y.Bytes()...)
	pubKeyHex := hex.EncodeToString(pubKeyBytes)

	jsonData := WalletJSON{
		PrivateKey: hex.EncodeToString(privBytes),
		PublicKey:  pubKeyHex,
		Address:    w.Address,
	}

	data, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal wallet: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0700); err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0600)
}

// Load wallet from JSON file
func LoadWallet(filePath string) (*Wallet, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var jsonData WalletJSON
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, fmt.Errorf("invalid wallet format: %w", err)
	}

	dBytes, err := hex.DecodeString(jsonData.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key encoding: %w", err)
	}

	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = elliptic.P256()
	priv.D = new(big.Int).SetBytes(dBytes)
	priv.PublicKey.X, priv.PublicKey.Y = priv.PublicKey.Curve.ScalarBaseMult(dBytes)

	return &Wallet{
		PrivateKey: priv,
		PublicKey:  &priv.PublicKey,
		Address:    jsonData.Address,
	}, nil
}

// Optional: Check if wallet file exists
func WalletExists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}
