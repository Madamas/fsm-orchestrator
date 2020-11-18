package main

import (
	"fmt"
	"github.com/Madamas/fsm-orchestrator/packages/fsm"
	"github.com/Madamas/fsm-orchestrator/packages/queue"
	"github.com/Madamas/fsm-orchestrator/packages/receiver"
	"github.com/Madamas/fsm-orchestrator/packages/storage"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"log"
	"os"
	"sync"
	"time"
)

func blankFunc(_ *fsm.ExecutionContext) (fsm.NodeName, error) {
	return "", errors.New("keq")
}

func sec(_ *fsm.ExecutionContext) (fsm.NodeName, error) {
	time.Sleep(time.Second * 10)
	return "Second", nil
}

func fourth(_ *fsm.ExecutionContext) (fsm.NodeName, error) {
	time.Sleep(time.Second * 10)
	return "Fourth", nil
}

func NewRedisPool() (*redis.Pool, error) {
	redisPool := redis.Pool{
		MaxActive: 2,
		MaxIdle:   5,
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

func main() {
	stepMap := fsm.NewStepMap()
	stepMap.AddStep("First", []fsm.NodeName{"Second", "Third"}, sec)
	stepMap.AddStep("Second", []fsm.NodeName{"Fourth", "Fifth"}, fourth)
	stepMap.AddStep("Third", []fsm.NodeName{"Sixth", "Seventh"}, blankFunc)

	mongo, err := storage.NewMongoStorage("localhost", "fsm", "sample_executor")

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	sm := sync.Map{}

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

	log.Println("Listening on 0.0.0.0:8086")
	log.Fatal(rec.ListenAndServe())
}
