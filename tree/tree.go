package tree

import (
	"fmt"
	"strconv"

	"github.com/RononDex/profilestotracesconnector/internal"
)

type Tree[T interface{}] struct {
	RootNode *Node[T]
}

func DumpTree(tree *Tree[internal.SampleLocation]) {
	currentNode := tree.RootNode
	dumpTreeRecursive(currentNode)
}

func dumpTreeRecursive(currentNode *Node[internal.SampleLocation]) {
	printValue("Node: "+currentNode.Value.Label, int(currentNode.Value.Level))
	printValue("-> ParentSpan: "+currentNode.Value.ParentSpanId.String(), int(currentNode.Value.Level))
	printValue("-> DurationInNs: "+strconv.Itoa(int(currentNode.Value.DurationInNs)), int(currentNode.Value.Level))
	printValue("-> Level: "+strconv.Itoa(int(currentNode.Value.Level)), int(currentNode.Value.Level))
	printValue("-> StartTimeStamp: "+currentNode.Value.StartTimeStamp.String(), int(currentNode.Value.Level))
	printValue("-> Self: "+strconv.Itoa(int(currentNode.Value.Self)), int(currentNode.Value.Level))
	for subNodeIdx := 0; subNodeIdx < len(currentNode.SubNodes); subNodeIdx++ {
		subNode := currentNode.SubNodes[subNodeIdx]
		dumpTreeRecursive(subNode)
	}
}

func printValue(value string, level int) {
	message := ""
	for levelIdx := 0; levelIdx < level; levelIdx++ {
		message += "--"
	}

	message += " "
	message += value
	fmt.Println(message)
}
