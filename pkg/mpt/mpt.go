package mpt

type MPT struct {
	Root Node
}

func NewMPT() *MPT {
	return &MPT{
		Root: NewBranchNode(),
	}
}

func (m *MPT) Insert(key []byte, value []byte) {
	nibbles := BytesToNibbles(key)
	m.Root = m.Root.Insert(nibbles, value)
}

func (m *MPT) Get(key []byte) ([]byte, bool) {
	nibbles := BytesToNibbles(key)
	return m.Root.Get(nibbles)
}

func (m *MPT) RootHash() []byte {
	return m.Root.Hash()
}

func (m *MPT) GenerateProof(key []byte) [][]byte {
	nibbles := BytesToNibbles(key)
	return m.Root.GenerateProof(nibbles)
}

func BuildMPTFromTxHashes(txHashes [][]byte) (*MPT, []byte) {
	trie := NewMPT()
	for _, h := range txHashes {
		trie.Insert(h, h) // Key = path from tx hash; Value = tx hash
	}
	return trie, trie.RootHash()
}
