package queue

import (
	"fmt"
	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"log"
)

const (
	queueNamespace     = "fsm-executor"
	QueueJobName       = "execute-function"
	HandlerConcurrency = 5
)

type Context struct {
	executorChannel chan<- string
}

func (c *Context) NotifyContext(id string) (err error) {
	defer func() {
		e := recover()
		if e != nil {
			err = errors.New(fmt.Sprintf("Couldn't notify executor: %v", e))
		}
	}()

	c.executorChannel <- id

	return
}

func (c *Context) Handle(job *work.Job) error {
	var globalErr error
	defer func() {
		err := recover()

		if err != nil {
			globalErr = errors.Errorf("Job handler panicked with error %v", fmt.Sprintf("%v", err))
		}

		if globalErr != nil {
			log.Println(globalErr.Error())
		}
	}()

	if _, ok := job.Args["jobId"]; ok {
		jobId := job.ArgString("jobId")
		if err := job.ArgError(); err != nil {
			globalErr = err
			return err
		}

		globalErr = c.NotifyContext(jobId)
	} else {
		globalErr = errors.New("Job ID can't be empty")
		return globalErr
	}

	return globalErr
}

func NewEnqueuer(pool *redis.Pool) *work.Enqueuer {
	return work.NewEnqueuer(queueNamespace, pool)
}

func NewHandler(executorChannel chan<- string, pool *redis.Pool) *work.WorkerPool {
	ctx := &Context{
		executorChannel: executorChannel,
	}

	wp := work.NewWorkerPool(*ctx, HandlerConcurrency, queueNamespace, pool)
	wp.JobWithOptions(QueueJobName, work.JobOptions{
		MaxFails: 1,
		SkipDead: true,
	}, ctx.Handle)

	return wp
}
