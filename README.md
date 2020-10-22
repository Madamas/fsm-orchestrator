# Finite-state machine orchestrator

Main conception is to create easy to use and deploy solution that aims to provide you means for functions pipeline execution of any complexity.  
This repository provides abstract api that implements execution graph storage and invocation via http and basic storage interface.

## Workflow
The main concept is providing your execution directional acyclic graph with transient functions. 
Before storing, each graph will be topographically sorted, checked for cycles and all roots will be found (if applicable).
**If your graph will contain more than one root then deepest will be selected.**  

At each cycle executor will check with which value your function returned and will proceed to next node in graph.
Each node implicitly has connection to error node which prematurely exits from the pipeline and writes error into provided storage.
If function returns value that isn't any connected node name then executor will stop pipeline and consider it finished.

## Usage
### Function dependency pass
Describe your function with handler signature
```go
func MyBestFunction(ec *fsm.ExecutionContext) (fsm.NodeName, error) {
    dep, ok := ec.ExecutionDependencies.Load("myDependencyKey")
    
    // sync map casts everything to interface
    // so cast your value to concrete type
    dep.(YourConcreteType).YourFieldOrFunction
    ...
    return "nextNodeName", nil
}
```
Then provide requested dependencies on executor startup
```go
sm := sync.Map{}
sm.Store("myDependencyKey", MySuperImportantDependency)
executor := fsm.NewExecutor(mongoStorage, sm)
```

### Function parameters pass
Refer to such function that uses param from execution context
```go
func MyBestFunction(ec *fsm.ExecutionContext) (fsm.NodeName, error) {
    val, ok := ec.Params["myParameterKey"]
}
```
Function parameters are passed into graph initiator via the receiver in next fashion.
Graph invocations are stored using your storage implementation, so you can review them later.
```json
{
  "graphName": "MyBestGraph",
  "params": {
    "myParameterKey": 1
  }
}
```

### Full usage example
```go
func blankFunc(_ *fsm.ExecutionContext) (fsm.NodeName, error) { return "", nil }

stepMap := fsm.NewStepMap()
sm.AddStep("First", []fsm.NodeName{"Second", "Third"}, blankFunc)
sm.AddStep("Second", []fsm.NodeName{"Fourth", "Fifth"}, blankFunc)
sm.AddStep("Third", []fsm.NodeName{"Sixth", "Seventh"}, blankFunc)

mongo := storage.NewMongoStorage("url", "db", "table")
sm := make(sync.Map)

executor := fsm.NewExecutor(mongoStorage, sm)
executor.AddControlGraph("SuperControlGraph", stepMap)

rec := receiver.CreateHttpListener(executor.ExecutorChannel, mongo)

go func() {
    err := rec.ListenAndServe()
    if err != nil {os.Exit(1)}
    
    executor.StartProcessing()
}
``` 
