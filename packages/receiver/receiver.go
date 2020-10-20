package receiver

import (
	"encoding/json"
	"fmt"
	"github.com/Madamas/fsm-orchestrator/packages/storage"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
)

type payload struct {
	GraphName string                 `json:"graphName"`
	Params    map[string]interface{} `json:"params"`
}

type HandleContext struct {
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

func (hc *HandleContext) ServeHTTP(r http.ResponseWriter, req *http.Request) {
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

func CreateHttpListener(executorChannel chan<- string, repository storage.Repository) http.Server {
	hc := HandleContext{
		executorChannel: executorChannel,
		repository:      repository,
	}

	return http.Server{
		Addr:    "0.0.0.0:8086",
		Handler: &hc,
	}
}
