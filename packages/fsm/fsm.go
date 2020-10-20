package fsm

import (
	"fmt"
	"github.com/Madamas/fsm-orchestrator/packages/storage"
	"github.com/pkg/errors"
	"sync"
)

type ExecutionContext struct {
	Params                map[string]interface{}
	ExecutionDependencies sync.Map

	step                  NodeName
	prevStep              NodeName
	jobId                 string
}

type StepFunction func(execCont *ExecutionContext) (NodeName, error)

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
	ExecutorChannel   <-chan string
	storage           storage.Repository
	executionStore    executionStore
	consumerSemaphore sync.WaitGroup
}

func NewExecutor(storage storage.Repository) *Executor {
	echan := make(chan string)
	store := make(map[string]storeEntry)

	return &Executor{
		ExecutorChannel:   echan,
		storage:           storage,
		consumerSemaphore: sync.WaitGroup{},
		executionStore: executionStore{
			store: store,
			mux:   sync.RWMutex{},
		},
	}
}

func (e *Executor) AddControlGraph(name string, sm stepMap) error {
	root, err := checkGraph(sm)

	if err != nil {
		return err
	}

	e.executionStore.storeGraph(name, storeEntry{
		stepMap: sm,
		root:    root,
	})

	return nil
}

func (e *Executor) StartProcessing() {
	defer e.consumerSemaphore.Wait()

	for i := 0; i < 5; i++ {
		e.consumerSemaphore.Add(1)
		go e.stepConsumer()
	}
}

func (e *Executor) executeGraph(node NodeName, al stepMap, execCont *ExecutionContext) error {
	// inability to checkin shouldn't cripple graph execution
	_ = e.storage.CheckinJob(execCont.jobId, string(node))
	executor, ok := al[node]

	if !ok {
		return nil
	}

	nextNode, err := executor.function(execCont)

	if err != nil {
		return err
	}

	execCont.prevStep = node
	execCont.step = nextNode

	return e.executeGraph(nextNode, al, execCont)
}

func (e *Executor) stepConsumer() {
	defer e.consumerSemaphore.Done()

	for event := range e.ExecutorChannel {
		job, err := e.storage.FindById(event)

		if err != nil || job == nil {
			fmt.Println("Couldn't find job", err)
		}

		graph, ok := e.executionStore.loadGraph(job.CommandGraph)

		if !ok {
			err = e.storage.FailJob(job.ID, errors.New("execution graph wasn't loaded"))
			if err != nil {
				fmt.Println("Couldn't fail job", err.Error())
			}
		}

		eCont := ExecutionContext{
			Params:   job.Params,
			step:     graph.root,
			prevStep: graph.root,
			jobId:    job.ID,
		}

		err = e.executeGraph(graph.root, graph.stepMap, &eCont)

		if err != nil {
			_ = e.storage.FailJob(job.ID, err)
		}
	}
}
