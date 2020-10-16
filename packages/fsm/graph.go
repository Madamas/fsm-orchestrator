package fsm

import (
	"fmt"
)

type VerticeName string

type VerticeTuple struct {
	From VerticeName
	To   VerticeName
}

type AdjacencyList map[VerticeName][]VerticeName

func goDeep(start VerticeName, al AdjacencyList, visited *nodeSet) bool {
	vertices, ok := al[start]

	if !ok {
		return false
	}

	(*visited)[start] = true

	for _, node := range vertices {
		_, alreadyWasThere := (*visited)[node]

		if alreadyWasThere {
			return true
		}

		hasCycle := goDeep(node, al, visited)

		if hasCycle {
			return true
		}
	}

	return false
}
// Kahn's algorithm
func IsAcyclic(al AdjacencyList) bool {
	ns := NewNodeSet()

	defer func(nodeset nodeSet) {
		fmt.Printf("\nVisited nodes %v\n", nodeset)
	}(ns)

	for node, _ := range al {
		hasCycle := goDeep(node, al, &ns)

		if hasCycle {
			return false
		}
	}

	return true
}
