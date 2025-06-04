package blockchain

import (
	"crypto/sha256"
)

// Tính Merkle Root từ danh sách transaction hash
func ComputeMerkleRoot(hashes [][]byte) []byte {

	if len(hashes) == 0 {
		return nil
	}
	if len(hashes) == 1 {
		return hashes[0]
	}

	var nextLevel [][]byte
	for i := 0; i < len(hashes); i += 2 {
		if i+1 == len(hashes) {
			// Nếu số lượng lẻ, duplicate hash cuối
			hashes = append(hashes, hashes[i])
		}
		combined := append(hashes[i], hashes[i+1]...)
		hash := sha256.Sum256(combined)
		nextLevel = append(nextLevel, hash[:])
	}
	return ComputeMerkleRoot(nextLevel)

}
