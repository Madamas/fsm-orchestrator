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

type Color string

var (
	Grey  Color = "grey"
	Black Color = "black"
)

type Colormap map[VerticeName]Color

func goDeep(start VerticeName, al AdjacencyList, nodeColors *Colormap, visitStack *[]VerticeName) bool {
	color := (*nodeColors)[start]
	if color == Black {
		return false
	}

	vertices, ok := al[start]

	if !ok {
		(*nodeColors)[start] = Black
		*visitStack = append(*visitStack, start)

		return false
	}

	color = (*nodeColors)[start]
	if color != Black {
		(*nodeColors)[start] = Grey
	}

	for _, node := range vertices {
		color, ok := (*nodeColors)[node]

		if ok {
			if color == Black {
				*visitStack = append(*visitStack, start)
			}

			return color == Grey
		}

		hasCycle := goDeep(node, al, nodeColors, visitStack)

		if hasCycle {
			return true
		}
	}

	color = (*nodeColors)[start]
	if color == Grey {
		(*nodeColors)[start] = Black
		*visitStack = append(*visitStack, start)
	}

	return false
}

// DFS algorithm
func IsAcyclic(al AdjacencyList) bool {
	cm := make(Colormap)
	vs := []VerticeName{}

	defer func() {
		fmt.Printf("\nVisited nodes %v\n", vs)
		fmt.Printf("\nNode colors %v\n", cm)
	}()

	for node := range al {
		hasCycle := goDeep(node, al, &cm, &vs)

		if hasCycle {
			return false
		}
	}

	return true
}
