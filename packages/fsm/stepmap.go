package fsm

// key is node we step into, value is it's children and function to be executed
type stepMap map[NodeName]nodeMap

func NewStepMap() stepMap {
	return make(stepMap)
}

func (sm stepMap) AddStep(node NodeName, childrenNodes []NodeName, nodeFunction StepFunction) {
	sm[node] = nodeMap{
		children: NewNodeSet(childrenNodes...),
		function: nodeFunction,
	}
}
