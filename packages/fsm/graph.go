package fsm

import (
	"errors"
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
func DfsSort(al AdjacencyList) (hasCycle bool, sortedNodes []VerticeName) {
	cm := make(Colormap)

	for node := range al {
		hasCycle = goDeep(node, al, &cm, &sortedNodes)

		if hasCycle {
			return
		}
	}

	return
}

func PlotAdjacencyList(sm StepMap) (AdjacencyList, []VerticeName, error) {
	al := make(AdjacencyList, len(sm))

	for k, v := range sm {
		_, ok := al[k]

		if !ok {
			al[k] = v.Children
		} else {
			al[k] = append(al[k], v.Children...)
		}
	}

	hasCycles, sortedGraph := DfsSort(al)

	if hasCycles {
		return AdjacencyList{}, []VerticeName{}, errors.New("control graph can't hold cycles")
	}

	return al, sortedGraph, nil
}
