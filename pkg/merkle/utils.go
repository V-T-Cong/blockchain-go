package merkle

import (
	"golang.org/x/crypto/sha3"
)

func BytesToNibbles(key []byte) []uint8 {

	nibbles := make([]uint8, 0, len(key)*2)
	for _, b := range key {
		nibbles = append(nibbles, b>>4, b&0x0F)
	}

	return nibbles
}

func CompactEncode(nibbles []uint8, isLeaf bool) []byte {

	flags := byte(0)

	if isLeaf {
		flags |= 0x20
	}

	if len(nibbles)%2 == 1 {
		flags |= 0x10 // odd length
		nibbles = append([]uint8{0}, nibbles...)
	}
	encoded := []byte{flags}

	for i := 0; i < len(nibbles); i += 2 {
		encoded = append(encoded, nibbles[i]<<4|nibbles[i+1])
	}

	return encoded
}

func Hash(data []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	return h.Sum(nil)
}
