package receiver

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type payload struct {
	GraphName string                 `json:"graphName"`
	Params    map[string]interface{} `json:"params"`
}

type HandleContext struct {
	eventChannel chan<- string
}

func (hc *HandleContext) ServeHTTP(r http.ResponseWriter, req *http.Request) {
	var payload payload

	data, err := ioutil.ReadAll(req.Body)

	if err != nil {
		r.WriteHeader(500)
		return
	}

	err = json.Unmarshal(data, &payload)

	if err != nil {
		r.Write([]byte(err.Error()))
		r.WriteHeader(400)
		return
	}

}

func StartHttpListener() http.Server {
	hc := HandleContext{
		eventChannel: make(chan string),
	}

	return http.Server{
		Addr:    "0.0.0.0:8086",
		Handler: &hc,
	}
}
