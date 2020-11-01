package receiver

import (
	"encoding/json"
	"fmt"
	"github.com/Madamas/fsm-orchestrator/packages/fsm"
	"github.com/Madamas/fsm-orchestrator/packages/storage"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
)

type payload struct {
	GraphName string                 `json:"graphName"`
	Params    map[string]interface{} `json:"params"`
}

type HandleContext struct {
	jobStack        fsm.JobStackLister
	executorChannel chan<- string
	repository      storage.Repository
}

func mapObjectDto(payload payload) storage.ObjectDTO {
	return storage.ObjectDTO{
		Status:       storage.Initial,
		CommandGraph: payload.GraphName,
		Params:       payload.Params,
	}
}

func (hc *HandleContext) createJob(r http.ResponseWriter, req *http.Request) {
	var payload payload

	body, err := ioutil.ReadAll(req.Body)

	if err != nil {
		r.WriteHeader(500)
		return
	}

	err = json.Unmarshal(body, &payload)

	if err != nil {
		r.Write([]byte(err.Error()))
		r.WriteHeader(400)
		return
	}

	obj, err := hc.repository.CreateJob(mapObjectDto(payload))

	if err != nil {
		r.Write([]byte(err.Error()))
		r.WriteHeader(500)
		return
	}

	err = hc.NotifyContext(obj.ID)

	if err != nil {
		r.Write([]byte(err.Error()))
		r.WriteHeader(500)
		hc.repository.FailJob(obj.ID, err)
		return
	}

	resp, err := json.Marshal(obj)

	if err != nil {
		r.Write([]byte(err.Error()))
		r.WriteHeader(500)
		hc.repository.FailJob(obj.ID, err)
		return
	}

	r.Write(resp)
	r.WriteHeader(200)

	return
}

func (hc *HandleContext) getJob(r http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	jobId := vars["id"]

	job, err := hc.repository.FindById(jobId)

	if err != nil {
		r.WriteHeader(http.StatusInternalServerError)
		r.Write([]byte(err.Error()))
		return
	}

	if data, err := json.Marshal(job); err != nil {
		r.WriteHeader(http.StatusInternalServerError)
		r.Write([]byte(err.Error()))
	} else {
		r.WriteHeader(http.StatusOK)
		r.Write(data)
	}
}

func (hc *HandleContext) listJobs(r http.ResponseWriter, req *http.Request) {
	jobs := hc.jobStack.ListJobs()

	data, err := json.Marshal(jobs)

	if err != nil {
		r.WriteHeader(http.StatusInternalServerError)
		r.Write([]byte(err.Error()))
	}

	r.WriteHeader(http.StatusOK)
	r.Write(data)
}

func (hc *HandleContext) NotifyContext(id string) (err error) {
	defer func() {
		e := recover()
		if e != nil {
			err = errors.New(fmt.Sprintf("Couldn't notify executor: %v", e))
		}
	}()

	hc.executorChannel <- id

	return
}

func CreateHttpListener(executorChannel chan<- string, repository storage.Repository, jobStack fsm.JobStackLister) http.Server {
	hc := HandleContext{
		jobStack:        jobStack,
		executorChannel: executorChannel,
		repository:      repository,
	}

	router := mux.NewRouter()
	router.HandleFunc("/jobs", hc.createJob).Methods("POST")
	router.HandleFunc("/jobs/list", hc.getJob).Methods("GET")
	router.HandleFunc("/jobs/{id}", hc.getJob).Methods("GET")

	return http.Server{
		Addr:    "0.0.0.0:8086",
		Handler: router,
	}
}
