package fsm

import (
	"fmt"
	"strings"
)

type nodeSet map[NodeName]bool

func NewNodeSet(vn ...NodeName) nodeSet {
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

func (ns nodeSet) SetMany(nodes ...NodeName) {
	for _, node := range nodes {
		ns.Set(node)
	}
}

func (ns nodeSet) Set(node NodeName) {
	_, ok := ns[node]

	if ok {
		return
	}

	ns[node] = true
}

func (ns nodeSet) Has(node NodeName) bool {
	_, ok := ns[node]

	return ok
}

func (ns nodeSet) AppendNodeSet(nodeset nodeSet) {
	for node := range nodeset {
		ns.Set(node)
	}
}