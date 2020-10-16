package fsm

import "errors"

type ExecutionContext struct {
	step                  VerticeName
	prevStep              VerticeName
	ExecutionDependencies map[string]interface{}
}

type StepFunction func(econt ExecutionContext) (VerticeName, error)

type NodeMap struct {
	Children []VerticeName
	Function StepFunction
}

func StepExecutor(sm StepMap, al AdjacencyList) {

}

func PlotAdjacencyList(sm StepMap) (AdjacencyList, error) {
	al := make(AdjacencyList, len(sm))

	for k, v := range sm {
		_, ok := al[k]

		if !ok {
			al[k] = v.Children
		} else {
			al[k] = append(al[k], v.Children...)
		}
	}

	hasCycles := IsAcyclic(al)

	if !hasCycles {
		return AdjacencyList{}, errors.New("control graph can't hold cycles")
	}

	return al, nil
}
