package fsm

// key is node we step into, value is it's children and function to be executed
type StepMap map[VerticeName]NodeMap

func NewStepMap() StepMap {
	return make(StepMap)
}

func (sm StepMap) AddStep(node VerticeName, childrenNodes []VerticeName, nodeFunction StepFunction) {
	sm[node] = NodeMap{
		Children: childrenNodes,
		Function: nodeFunction,
	}
}
