package receiver

import (
	"encoding/json"
	"fmt"
	"github.com/Madamas/fsm-orchestrator/packages/fsm"
	"github.com/Madamas/fsm-orchestrator/packages/queue"
	"github.com/Madamas/fsm-orchestrator/packages/storage"
	"github.com/gocraft/work"
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
	jobStack   fsm.JobStackLister
	enqueuer   *work.Enqueuer
	repository *storage.Repository
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
		r.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(body, &payload)

	if err != nil {
		r.WriteHeader(http.StatusBadRequest)
		r.Write([]byte(err.Error()))
		return
	}

	obj, err := hc.repository.CreateJob(mapObjectDto(payload))

	if err != nil {
		r.WriteHeader(http.StatusInternalServerError)
		r.Write([]byte(err.Error()))
		return
	}

	err = hc.NotifyContext(obj.ID.(string))

	if err != nil {
		r.WriteHeader(http.StatusInternalServerError)
		r.Write([]byte(err.Error()))
		hc.repository.FailJob(obj.ID.(string), err)
		return
	}

	resp, err := json.Marshal(obj)

	if err != nil {
		r.WriteHeader(http.StatusInternalServerError)
		r.Write([]byte(err.Error()))
		hc.repository.FailJob(obj.ID.(string), err)
		return
	}

	r.Header().Set("Content-Type", "application/json")
	r.WriteHeader(http.StatusOK)
	r.Write(resp)

	return
}

func (hc *HandleContext) getJob(r http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	jobId := vars["id"]

	job, err := hc.repository.FindById(jobId)

	if err != nil {
		fmt.Println("govno")
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

	_, err = hc.enqueuer.Enqueue(queue.QueueJobName, work.Q{
		"jobId": id,
	})

	return
}

func CreateHttpListener(enqueuer *work.Enqueuer, repository *storage.Repository, jobStack fsm.JobStackLister) http.Server {
	hc := HandleContext{
		jobStack:   jobStack,
		enqueuer:   enqueuer,
		repository: repository,
	}

	router := mux.NewRouter()
	router.HandleFunc("/jobs", hc.createJob).Methods("POST")
	router.HandleFunc("/jobs/list", hc.listJobs).Methods("GET")
	router.HandleFunc("/jobs/{id}", hc.getJob).Methods("GET")

	return http.Server{
		Addr:    "0.0.0.0:8086",
		Handler: router,
	}
}
