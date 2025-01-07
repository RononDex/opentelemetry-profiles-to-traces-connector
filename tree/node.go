package tree

type Node[T interface{}] struct {
	SubNodes []Node[T]
	Value    T
}
