package tree

import "fmt"

type Tree[T interface{}] struct {
	RootNode Node[T]
}

func DumpTree[T interface{}](tree *Tree[T]) {
	currentNode := tree.RootNode
	dumpTreeRecursive(&currentNode)
}

func dumpTreeRecursive[T interface{}](currentNode *Node[T]) {
	fmt.Println(currentNode)
	for subNodeIdx := 0; subNodeIdx < len(currentNode.SubNodes); subNodeIdx++ {
		subNode := currentNode.SubNodes[subNodeIdx]
		dumpTreeRecursive(&subNode)
	}
}
