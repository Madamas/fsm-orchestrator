package fsm

import (
	"github.com/pkg/errors"
)

type NodeName string

type VerticeTuple struct {
	From NodeName
	To   NodeName
}

type color string

var (
	Grey  color = "grey"
	Black color = "black"
)

type colormap map[NodeName]color

func deepSearch(start NodeName, sm stepMap, nodeColors *colormap, children nodeSet, roots nodeSet) bool {
	if !children.Has(start) {
		roots.Set(start)
	}

	color := (*nodeColors)[start]
	if color == Black {
		return false
	}

	node, ok := sm[start]

	if !ok {
		(*nodeColors)[start] = Black
		return false
	}

	color = (*nodeColors)[start]
	if color != Black {
		(*nodeColors)[start] = Grey
	}

	for node, _ := range node.children {
		color, ok := (*nodeColors)[node]

		if ok {
			if color == Black {
				(*nodeColors)[start] = Black
			}

			return color == Grey
		}

		hasCycle := deepSearch(node, sm, nodeColors, children, roots)

		if hasCycle {
			return true
		}
	}

	color = (*nodeColors)[start]
	if color == Grey {
		(*nodeColors)[start] = Black
	}

	return false
}

func probeDepth(parent NodeName, sm stepMap) int {
	node, ok := sm[parent]
	if !ok {
		return 1
	}

	max := 1

	for node, _ := range node.children {
		val := 1 + probeDepth(node, sm)
		if val > max {
			max = val
		}
	}

	return max
}

// DfsSort uses recursive deep first search algorithm to find cycles
// and topologically sort control graph
func dfsSort(sm stepMap, childrens nodeSet) (hasCycle bool, roots nodeSet) {
	cm := make(colormap)
	roots = NewNodeSet()

	for node := range sm {
		hasCycle = deepSearch(node, sm, &cm, childrens, roots)

		if hasCycle {
			return
		}
	}

	return
}

// checkGraph finds if control graph and finds deepest root
func checkGraph(sm stepMap) (NodeName, error) {
	childrenList := NewNodeSet()

	for _, v := range sm {
		childrenList.AppendNodeSet(v.children)
	}

	hasCycles, roots := dfsSort(sm, childrenList)

	if hasCycles {
		return "", errors.New("control graph can't hold cycles")
	}

	var maxRootName NodeName
	maxRootDepth := 1

	for node, _ := range roots {
		depth := probeDepth(node, sm)
		if depth >= maxRootDepth {
			maxRootDepth = depth
			maxRootName = node
		}
	}

	return maxRootName, nil
}
