package main

import (
	"fmt"

	"github.com/Madamas/fsm-orchestrator/packages/fsm"
)

func blankFunc(_ fsm.ExecutionContext) (fsm.NodeName, error) { return "", nil }

func main() {
	sm := fsm.NewStepMap()
	sm.AddStep("First", []fsm.NodeName{"Second", "Third"}, blankFunc)
	sm.AddStep("Second", []fsm.NodeName{"Fourth", "Fifth"}, blankFunc)
	sm.AddStep("Third", []fsm.NodeName{"Sixth", "Seventh"}, blankFunc)
	sm.AddStep("Eight", []fsm.NodeName{"Sixth", "Seventh"}, blankFunc)

	root, err := fsm.CheckGraph(sm)

	if err != nil {
		fmt.Print(err.Error())
	}

	fmt.Println("AdjList", sm)
	fmt.Println("Root", root)
}
