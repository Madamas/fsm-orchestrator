package storage

import (
	"time"

	"github.com/Madamas/fsm-orchestrator/packages/fsm"
)

type Status string

var (
	Initial    Status = "initial"
	Processing Status = "processing"
	Completed  Status = "completed"
	Failed     Status = "failed"
)

type Object struct {
	ID        string    `bson:"_id" json:"id"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
	Error     string    `bson:"error" json:"error"`
	ObjectDTO
}

type ObjectDTO struct {
	CurrentStep  fsm.VerticeName `bson:"currentStep" json:"currentStep"`
	CommandGraph string          `bson:"commandGraph" json:"commandGraph"`
	Status       Status          `bson:"status" json:"status"`
}

type KV map[string]interface{}

type Storage interface {
	Create(obj ObjectDTO) (Object, error)
	FindById(id string) (Object, error)
	UpdateById(id string, update KV) error
}
