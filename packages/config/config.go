package config

import (
	"github.com/Madamas/fsm-orchestrator/packages/fsm"
	"github.com/Madamas/fsm-orchestrator/packages/storage"
	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
	"time"
)

type MongodbConfig struct {
	Url              string
	Database         string
	Table            string
	ReconnectTimeout time.Duration
}

type HttpListener struct {
	Enqueuer     *work.Enqueuer
	Repository   *storage.Repository
	JobStack     fsm.JobStackLister
	QueueJobName string
}

type Enqueuer struct {
	QueueNamespace string
	RedisPool      *redis.Pool
}

type Handler struct {
	QueueNamespace  string
	QueueJobName    string
	Concurrency     uint
	ExecutorChannel chan<- string
	RedisPool       *redis.Pool
}
