package fsm

import (
	"errors"
	"fmt"
	"github.com/Madamas/fsm-orchestrator/packages/storage"
	"sync"
)

type ExecutionContext struct {
	step                  NodeName
	prevStep              NodeName
	ExecutionDependencies map[string]interface{}
}

type StepFunction func(econt ExecutionContext) (NodeName, error)

type nodeMap struct {
	children nodeSet
	function StepFunction
}

type storeEntry struct {
	stepMap stepMap
	root    NodeName
}

type executionStore struct {
	mux   sync.RWMutex
	store map[string]storeEntry
}

func (es *executionStore) loadGraph(key string) (val storeEntry, ok bool) {
	es.mux.RLock()
	defer es.mux.RUnlock()

	val, ok = es.store[key]
	return
}

func (es *executionStore) storeGraph(key string, entry storeEntry) {
	es.mux.Lock()
	defer es.mux.Unlock()

	es.store[key] = entry
}

type Executor struct {
	executorChannel <-chan string
	storage         storage.Repository

	executionStore executionStore

	consumerSemaphore sync.WaitGroup
}

func (e *Executor) AddControlGraph(name string, sm stepMap) error {
	root, err := CheckGraph(sm)

	if err != nil {
		return err
	}

	e.executionStore.storeGraph(name, storeEntry{
		stepMap: sm,
		root:    root,
	})

	return nil
}
func (e *Executor) executeGraph(root NodeName, al stepMap, params map[string]interface{}) {

}
func (e *Executor) stepConsumer() {
	defer e.consumerSemaphore.Done()

	for event := range e.executorChannel {
		job, err := e.storage.FindById(event)

		if err != nil {
			fmt.Println("Couldn't find job", err.Error())
		}

		graph, ok := e.executionStore.loadGraph(job.CommandGraph)

		if !ok {
			err = e.storage.FailJob(job.ID, errors.New("execution graph wasn't loaded"))
			if err != nil {
				fmt.Println("Couldn't fail job", err.Error())
			}
		}

		e.executeGraph(graph.root, graph.stepMap, job.Params)
	}
}
func (e *Executor) StartProcessing() {
	defer e.consumerSemaphore.Wait()

	for i := 0; i < 5; i++ {
		e.consumerSemaphore.Add(1)
	}
}
