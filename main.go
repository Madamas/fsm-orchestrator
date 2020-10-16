package main

import (
	"fmt"
	"fsm/packages/fsm"
)

func blankFunc(_ fsm.ExecutionContext) (fsm.VerticeName, error) {return "", nil}

func main() {
	sm := fsm.NewStepMap()
	sm.AddStep("First", []fsm.VerticeName{"Second", "Third"}, blankFunc)
	sm.AddStep("Second", []fsm.VerticeName{"Fourth", "Fifth"}, blankFunc)
	sm.AddStep("Third", []fsm.VerticeName{"Sixth", "Seventh"}, blankFunc)

	_, err := fsm.PlotAdjacencyList(sm)

	if err != nil {
		fmt.Print(err.Error())
	}
}
