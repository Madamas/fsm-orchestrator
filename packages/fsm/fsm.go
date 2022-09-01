package fsm

import (
	"fmt"
	"log"
	"sync"

	"github.com/Madamas/fsm-orchestrator/packages/storage"
	"github.com/pkg/errors"
)

type ExecutionContext struct {
	Params                map[string]interface{}
	ExecutionDependencies *sync.Map

	step     NodeName
	prevStep NodeName
	JobId    string
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
	ExecutorChannel       chan string
	JobStack              JobStack
	executionDependencies *sync.Map
	storage               *storage.Repository
	executionStore        executionStore
	consumerSemaphore     sync.WaitGroup
	concurrency           int
}

func NewExecutor(storage *storage.Repository, dependencies *sync.Map, concurrency int) *Executor {
	echan := make(chan string)
	store := make(map[string]storeEntry)
	jobStack := NewJobStack(5)

	return &Executor{
		ExecutorChannel:       echan,
		JobStack:              jobStack,
		executionDependencies: dependencies,
		storage:               storage,
		consumerSemaphore:     sync.WaitGroup{},
		concurrency:           concurrency,
		executionStore: executionStore{
			store: store,
			mux:   sync.RWMutex{},
		},
	}
}

func (e *Executor) GetJobStack() JobStackLister {
	return e.JobStack
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

	for i := 0; i < e.concurrency; i++ {
		e.consumerSemaphore.Add(1)
		go e.stepConsumer()
	}
}

func (e *Executor) executeGraph(node NodeName, al stepMap, execCont *ExecutionContext) error {
	// inability to checkin shouldn't cripple graph execution
	err := e.storage.CheckinJob(execCont.JobId, string(node))
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
	var previousEvent string

	for event := range e.ExecutorChannel {
		// costyl for in-loop defer
		if previousEvent != "" {
			e.JobStack.FinishJob(previousEvent)
			log.Printf("Finished event %s", previousEvent)
			previousEvent = ""
		}

		e.JobStack.StartJob(event)
		log.Printf("Started event %s", event)
		previousEvent = event

		job, err := e.storage.FindById(event)

		if err != nil || job == nil {
			fmt.Println("Couldn't find job", err)
			continue
		}

		graph, ok := e.executionStore.loadGraph(job.CommandGraph)

		if !ok {
			err = e.storage.FailJob(job.ID.(string), errors.New("execution graph wasn't loaded"))
			if err != nil {
				fmt.Println("Couldn't fail job", err.Error())
			}
		}

		err = e.storage.StartJob(job.ID.(string), string(graph.root))
		if err != nil {
			fmt.Println("Error while starting job", err)
			continue
		}

		eCont := ExecutionContext{
			Params:                job.Params,
			ExecutionDependencies: e.executionDependencies,
			step:                  graph.root,
			prevStep:              graph.root,
			JobId:                 job.ID.(string),
		}

		err = e.executeGraph(graph.root, graph.stepMap, &eCont)

		if err != nil {
			_ = e.storage.FailJob(job.ID.(string), err)
		} else {
			_ = e.storage.CompleteJob(job.ID.(string))
		}
	}
}
