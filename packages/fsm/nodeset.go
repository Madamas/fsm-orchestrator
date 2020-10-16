package fsm

import (
	"fmt"
	"strings"
)

type nodeSet map[VerticeName]bool

func NewNodeSet(vn ...VerticeName) nodeSet {
	ns := make(nodeSet, len(vn))

	for _, v := range vn {
		ns[v] = true
	}

	return ns
}

func (ns nodeSet) String() string {
	keys := make([]string, len(ns))

	i := 0
	for k,_ := range ns {
		keys[i] = string(k)
		i++
	}

	keysJoined := strings.Join(keys, " ")

	return fmt.Sprintf("[%s]", keysJoined)
}

func (ns nodeSet) Set(node VerticeName) {
	_, ok := ns[node]

	if ok {
		return
	}

	ns[node] = true
}

func (ns nodeSet) Has(node VerticeName) bool {
	_, ok := ns[node]

	return ok
}
