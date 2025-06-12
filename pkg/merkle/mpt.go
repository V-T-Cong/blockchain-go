package merkle

type MPT struct {
	Root Node
}

func NewMPT() *MPT {
	return &MPT{}
}

func (t *MPT) Insert(key string, value []byte) {
	nibbles := BytesToNibbles([]byte(key))
	t.Root = insert(t.Root, nibbles, value)
}

func insert(node Node, nibbles []uint8, value []byte) Node {
	panic("unimplemented")
}
