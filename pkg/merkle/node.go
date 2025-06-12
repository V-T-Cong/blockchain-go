package merkle

type Node interface {
	Hash() []byte
}

type LeafNode struct {
	Path  []uint8
	Value []byte
}

type ExtensionNode struct {
	Path []uint8
	Next Node
}

type BrachNode struct {
	Branches [16]Node
	Value    []byte
}
