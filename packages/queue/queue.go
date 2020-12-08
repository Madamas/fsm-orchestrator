package queue

import (
	"fmt"
	"github.com/Madamas/fsm-orchestrator/packages/config"
	"github.com/gocraft/work"
	"github.com/pkg/errors"
	"log"
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

func NewEnqueuer(config config.Enqueuer) *work.Enqueuer {
	// TODO: add config validation
	return work.NewEnqueuer(config.QueueNamespace, config.RedisPool)
}

func NewHandler(config config.Handler) *work.WorkerPool {
	// TODO: add config validation
	ctx := &Context{
		executorChannel: config.ExecutorChannel,
	}

	wp := work.NewWorkerPool(*ctx, config.Concurrency, config.QueueNamespace, config.RedisPool)
	wp.JobWithOptions(config.QueueJobName, work.JobOptions{
		MaxFails: 1,
		SkipDead: true,
	}, ctx.Handle)

	return wp
}
