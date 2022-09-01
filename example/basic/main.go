package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/Madamas/fsm-orchestrator/packages/config"
	"github.com/Madamas/fsm-orchestrator/packages/fsm"
	"github.com/Madamas/fsm-orchestrator/packages/queue"
	"github.com/Madamas/fsm-orchestrator/packages/receiver"
	"github.com/Madamas/fsm-orchestrator/packages/storage"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
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

	mongo, err := storage.NewMongoStorage(config.MongodbConfig{
		Url: "localhost",
		Database: "fsm",
		Table: "sample_executor",
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	sm := sync.Map{}

	executor := fsm.NewExecutor(mongo, &sm, 10)
	err = executor.AddControlGraph("SuperControlGraph", stepMap)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	rp, err := NewRedisPool()
	eq := queue.NewEnqueuer(config.Enqueuer{
		QueueNamespace: "test",
		RedisPool: rp,
	})
	handler := queue.NewHandler(config.Handler{
		QueueNamespace: "test",
		QueueJobName: "super_job",
	})
	handler.Start()
	defer func() {
		handler.Drain()
		handler.Stop()
	}()

	rec := receiver.CreateHttpListener(config.HttpListener{
		Enqueuer: eq,
		Repository: mongo,
		JobStack: executor.GetJobStack(),
		QueueJobName: "super_job",
	})
	go executor.StartProcessing()

	log.Println("Listening on 0.0.0.0:8086")
	log.Fatal(rec.ListenAndServe())
}
