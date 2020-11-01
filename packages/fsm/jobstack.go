package fsm

import "sync"

type jobStack struct {
	mux sync.RWMutex
	store map[string]bool
	lastView []string
}

type JobStackManipulator interface {
	StartJob(string)
	FinishJob(string)
}

type JobStackLister interface {
	ListJobs() []string
}

type JobStack interface {
	JobStackManipulator
	JobStackLister
}

func NewJobStack(concurrency int) JobStack {
	js := jobStack{
		mux: sync.RWMutex{},
		store: make(map[string]bool, concurrency),
		lastView: nil,
	}

	return &js
}

func (js *jobStack) StartJob(job string) {
	js.mux.Lock()
	defer js.mux.Unlock()

	js.store[job] = true
	js.lastView = nil
}

func (js *jobStack) FinishJob(job string) {
	js.mux.Lock()
	defer js.mux.Unlock()

	delete(js.store, job)
	js.lastView = nil
}

func (js *jobStack) ListJobs() []string {
	js.mux.RLock()
	defer js.mux.RUnlock()

	if js.lastView != nil {
		return js.lastView
	}
	// possible last view population race condition
	// think about how to mitigate it (maybe remove cached view altogether?)
	result := make([]string, 0, len(js.store))

	for job := range js.store {
		result = append(result, job)
	}

	js.lastView = result
	return result
}