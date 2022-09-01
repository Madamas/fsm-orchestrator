package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	"github.com/Madamas/fsm-orchestrator/packages/fsm"
	"github.com/Madamas/fsm-orchestrator/packages/queue"
	"github.com/Madamas/fsm-orchestrator/packages/receiver"
	"github.com/Madamas/fsm-orchestrator/packages/storage"
)

type FaasResponse struct {
    Status    string `json:"status"`
    StartTime int64  `json:"startTime"`
    ExitTime  int64  `json:"exitTime"`
}

type Bench struct {
    JobId     string
    FuncStart time.Time
    FaasStart time.Time
    FaasEnd   time.Time
    FuncEnd   time.Time
}

type MoreBench struct {
    Bench
    StartDiff time.Duration
    EndDiff   time.Duration
}

var glChan chan Bench
var datas []MoreBench

func init() {
    glChan = make(chan Bench, 10000)
    datas = make([]MoreBench, 10000)

    go func() {
        for val := range glChan {
            datas = append(datas, MoreBench{
                Bench:     val,
                StartDiff: val.FaasStart.Sub(val.FuncStart),
                EndDiff:   val.FuncEnd.Sub(val.FaasEnd),
            })
        }
    }()
}

func executor(client *http.Client) (*FaasResponse, error) {
    req, err := http.NewRequest("POST", "http://127.0.0.1:9091/function/fn1", nil)

    if err != nil {
        return nil, errors.Wrap(err, "errored while creating request")
    }

    res, err := client.Do(req)

    if err != nil {
        return nil, errors.Wrap(err, "request err")
    }
    defer res.Body.Close()
    data, err := ioutil.ReadAll(res.Body)

    if err != nil {
        return nil, errors.Wrap(err, "couldn't read response body")
    }
    var resp FaasResponse
    err = json.Unmarshal(data, &resp)

    if err != nil {
        return nil, errors.Wrap(err, "unmarshal err")
    }

    return &resp, nil
}
	
func first(ec *fsm.ExecutionContext) (fsm.NodeName, error) {
    client, ok := ec.ExecutionDependencies.Load("client")

    if !ok {
        client = http.DefaultClient
    }

    functionStart := time.Now()
    resp, err := executor(client.(*http.Client))
    functionEnd := time.Now()

    if err != nil {
        fmt.Println("Errored", err.Error())
        return "", err
    }
    glChan <- Bench{
        JobId:     ec.JobId,
        FuncStart: functionStart,
        FaasStart: time.Unix(0, resp.StartTime*int64(time.Millisecond)),
        FaasEnd:   time.Unix(0, resp.ExitTime*int64(time.Millisecond)),
        FuncEnd:   functionEnd,
    }
    return "Second", nil
}

func second(ec *fsm.ExecutionContext) (fsm.NodeName, error) {
    client, ok := ec.ExecutionDependencies.Load("client")

    if !ok {
        client = http.DefaultClient
    }
    functionStart := time.Now()
    resp, err := executor(client.(*http.Client))
    functionEnd := time.Now()

    if err != nil {
        return "", err
    }

    glChan <- Bench{
        JobId:     ec.JobId,
        FuncStart: functionStart,
        FaasStart: time.Unix(0, resp.StartTime*int64(time.Millisecond)),
        FaasEnd:   time.Unix(0, resp.ExitTime*int64(time.Millisecond)),
        FuncEnd:   functionEnd,
    }

    return "Third", nil
}

func third(ec *fsm.ExecutionContext) (fsm.NodeName, error) {
    client, ok := ec.ExecutionDependencies.Load("client")

    if !ok {
        client = http.DefaultClient
    }

    functionStart := time.Now()
    resp, err := executor(client.(*http.Client))
    functionEnd := time.Now()

    if err != nil {
        return "", err
    }

    glChan <- Bench{
        JobId:     ec.JobId,
        FuncStart: functionStart,
        FaasStart: time.Unix(0, resp.StartTime*int64(time.Millisecond)),
        FaasEnd:   time.Unix(0, resp.ExitTime*int64(time.Millisecond)),
        FuncEnd:   functionEnd,
    }

    fmt.Println("end")
    return "", nil
}

func NewRedisPool() (*redis.Pool, error) {
    redisPool := redis.Pool{
        MaxActive: 100,
        MaxIdle:   101,
        Wait:      true,
        Dial: func() (redis.Conn, error) {
            return redis.Dial("tcp", "localhost:6379")
        },
        TestOnBorrow: func(c redis.Conn, t time.Time) error {
            if time.Since(t) < time.Minute {
                return nil
            }
            _, err := c.Do("PING")
            return err
        },
    }

    conn := redisPool.Get()
    defer conn.Close()

    _, err := conn.Do("PING")

    if err != nil {
        return nil, errors.Wrap(err, "Ping command failed")
    }

    return &redisPool, nil
}

func writer() {
    router := mux.NewRouter()
    router.HandleFunc("/write", func(responseWriter http.ResponseWriter, request *http.Request) {
        file, err := os.Create("./report.csv")

        if err == nil {
            fmt.Println("file opened")
            for _, value := range datas {
                if !value.FuncStart.IsZero() {
                    _, err := file.WriteString(fmt.Sprintf(
                        "%d.%d,%d.%d,%d.%d,%d.%d,%d,%d\n",
                        value.FuncStart.Unix(),
                        value.FuncStart.Nanosecond(),
                        value.FaasStart.Unix(),
                        value.FaasStart.Nanosecond(),
                        value.FaasEnd.Unix(),
                        value.FaasEnd.Nanosecond(),
                        value.FuncEnd.Unix(),
                        value.FuncEnd.Nanosecond(),
                        value.StartDiff.Nanoseconds(),
                        value.EndDiff.Nanoseconds(),
                    ))

                    if err != nil {
                        fmt.Println("write err", err)
                    }
                }
            }

            _ = file.Close()
            fmt.Println("wrote")
        }
    }).Methods("GET")

    server := http.Server{
        Addr:    "0.0.0.0:8087",
        Handler: router,
    }

    log.Fatal(server.ListenAndServe())
}

func main() {
    stepMap := fsm.NewStepMap()
    stepMap.AddStep("First", []fsm.NodeName{"Second"}, first)
    stepMap.AddStep("Second", []fsm.NodeName{"Third"}, second)
    stepMap.AddStep("Third", []fsm.NodeName{}, third)

    mongo, err := storage.NewMongoStorage("localhost", "fsm", "test_executor")

    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }

    sm := sync.Map{}
    sm.Store("client", http.DefaultClient)

    executor := fsm.NewExecutor(mongo, &sm)
    err = executor.AddControlGraph("SuperControlGraph", stepMap)

    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }

    rp, err := NewRedisPool()
    eq := queue.NewEnqueuer(rp)
    handler := queue.NewHandler(executor.ExecutorChannel, rp)
    handler.Start()
    defer func() {
        handler.Drain()
        handler.Stop()
    }()

    rec := receiver.CreateHttpListener(eq, mongo, executor.GetJobStack())
    go executor.StartProcessing()
    go writer()

    log.Println("Listening on 0.0.0.0:8086")
    log.Fatal(rec.ListenAndServe())
}
