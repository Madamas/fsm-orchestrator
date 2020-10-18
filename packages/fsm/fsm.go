package fsm

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

func stepExecutor(sm StepMap, al AdjacencyList) {

}
