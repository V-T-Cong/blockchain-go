package mpt

import "crypto/sha256"

type Node interface {
	Hash() []byte
	Insert([]byte, []byte) Node
	Get(path []byte) ([]byte, bool)
	GenerateProof([]byte) [][]byte
}

type LeafNode struct {
	Key   []byte // Nibbles
	Value []byte
}

func (l *LeafNode) Hash() []byte {
	data := append(append([]byte{0x00}, l.Key...), l.Value...)
	h := sha256.Sum256(data)
	return h[:]
}

func (l *LeafNode) Insert(path []byte, value []byte) Node {
	// Replace this with full split logic if needed
	if Equal(path, l.Key) {
		l.Value = value
		return l
	}
	branch := NewBranchNode()
	branch.Insert(l.Key, l.Value)
	return branch.Insert(path, value)
}

func (l *LeafNode) Get(path []byte) ([]byte, bool) {
	if Equal(path, l.Key) {
		return l.Value, true
	}
	return nil, false
}

func (l *LeafNode) GenerateProof(path []byte) [][]byte {
	if Equal(path, l.Key) {
		return [][]byte{l.Hash()}
	}
	return nil
}

type BranchNode struct {
	Branches [16]Node
	Value    []byte // Optional value at this node
}

func NewBranchNode() *BranchNode {
	return &BranchNode{}
}

func (b *BranchNode) Hash() []byte {
	data := []byte{0x01}
	for _, n := range b.Branches {
		if n != nil {
			data = append(data, n.Hash()...)
		} else {
			data = append(data, make([]byte, 32)...) // empty hash
		}
	}
	if b.Value != nil {
		data = append(data, b.Value...)
	}
	h := sha256.Sum256(data)
	return h[:]
}

func (b *BranchNode) Insert(path []byte, value []byte) Node {
	if len(path) == 0 {
		b.Value = value
		return b
	}
	index := path[0]
	child := b.Branches[index]
	if child == nil {
		b.Branches[index] = &LeafNode{
			Key:   path[1:], // Rest of the path
			Value: value,
		}
	} else {
		b.Branches[index] = child.Insert(path[1:], value)
	}
	return b
}

func (b *BranchNode) Get(path []byte) ([]byte, bool) {
	if len(path) == 0 {
		if b.Value != nil {
			return b.Value, true
		}
		return nil, false
	}
	index := path[0]
	child := b.Branches[index]
	if child != nil {
		return child.Get(path[1:])
	}
	return nil, false
}

func (b *BranchNode) GenerateProof(path []byte) [][]byte {
	var proof [][]byte
	proof = append(proof, b.Hash())
	if len(path) == 0 {
		return proof
	}
	index := path[0]
	child := b.Branches[index]
	if child != nil {
		childProof := child.GenerateProof(path[1:])
		if childProof != nil {
			proof = append(proof, childProof...)
		}
	}
	return proof
}
