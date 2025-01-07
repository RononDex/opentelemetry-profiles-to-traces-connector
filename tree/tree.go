package tree

type Tree[T interface{}] struct {
	RootNode Node[T]
}
