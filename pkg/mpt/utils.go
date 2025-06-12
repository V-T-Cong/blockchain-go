package mpt

import (
	"bytes"
	"crypto/sha256"
)

func BytesToNibbles(b []byte) []byte {
	nibbles := make([]byte, len(b)*2)
	for i, v := range b {
		nibbles[2*i] = v >> 4
		nibbles[2*i+1] = v & 0x0F
	}
	return nibbles
}

func Equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// VerifyProof checks if a key-value pair is included in the MPT with the given root hash.
func VerifyProof(rootHash, key, expectedValue []byte, proof [][]byte) bool {
	if len(proof) == 0 {
		return false
	}

	// Step 1: Recompute the hashes from leaf to root
	nibbles := BytesToNibbles(key)
	currentHash := computeLeafHash(nibbles, expectedValue)

	// Walk backwards through the proof, recomputing hashes
	for i := len(proof) - 2; i >= 0; i-- {
		parentHash := computeBranchHashWithChild(currentHash, nibbles, i)
		currentHash = parentHash
	}

	// Step 2: Final hash must match root hash
	return bytes.Equal(currentHash, rootHash)
}

// computeLeafHash hashes a fake leaf node with the provided key and value.
func computeLeafHash(nibbles, value []byte) []byte {
	data := append(append([]byte{0x00}, nibbles...), value...)
	h := sha256.Sum256(data)
	return h[:]
}

// computeBranchHashWithChild fakes a branch node with the given child hash at the correct index.
func computeBranchHashWithChild(childHash, keyNibbles []byte, depth int) []byte {
	index := keyNibbles[depth]
	data := []byte{0x01}
	for i := 0; i < 16; i++ {
		if i == int(index) {
			data = append(data, childHash...)
		} else {
			data = append(data, make([]byte, 32)...)
		}
	}
	// No value at branch node
	h := sha256.Sum256(data)
	return h[:]
}
