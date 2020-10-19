package receiver

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Madamas/fsm-orchestrator/packages/storage"
	"io/ioutil"
	"net/http"
)

type payload struct {
	GraphName string                 `json:"graphName"`
	Params    map[string]interface{} `json:"params"`
}

type HandleContext struct {
	eventChannel chan<- string
	storage      storage.Storage
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

	obj, err := hc.CreateJob(payload)

	if err != nil {
		r.Write([]byte(err.Error()))
		r.WriteHeader(500)
		return
	}

	err = hc.NotifyContext(obj.ID)

	if err != nil {
		r.Write([]byte(err.Error()))
		r.WriteHeader(500)
		hc.FailJob(obj.ID, err)
		return
	}

	resp, err := json.Marshal(obj)

	if err != nil {
		r.Write([]byte(err.Error()))
		r.WriteHeader(500)
		hc.FailJob(obj.ID, err)
		return
	}

	r.Write(resp)
	r.WriteHeader(200)

	return
}

func (hc *HandleContext) CreateJob(payload payload) (storage.Object, error) {
	obj := storage.ObjectDTO{
		Status:       storage.Initial,
		CommandGraph: payload.GraphName,
	}

	return hc.storage.Create(obj)
}

func (hc *HandleContext) FailJob(id string, err error) {
	data := storage.KV{
		"status": storage.Failed,
		"error": err.Error(),
	}

	hc.storage.UpdateById(id, data)

	return
}

func (hc *HandleContext) NotifyContext(id string) (err error) {
	defer func() {
		e := recover()
		if e != nil {
			err = errors.New(fmt.Sprintf("Couldn't notify executor: %v", e))
		}
	}()

	hc.eventChannel <- id

	return
}

func CreateHttpListener(executorChannel chan<-string, storage storage.Storage) http.Server {
	hc := HandleContext{
		eventChannel: executorChannel,
		storage: storage,
	}

	return http.Server{
		Addr:    "0.0.0.0:8086",
		Handler: &hc,
	}
}
