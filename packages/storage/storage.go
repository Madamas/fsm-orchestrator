package storage

import (
	"time"

	"github.com/Madamas/fsm-orchestrator/packages/fsm"
	"github.com/globalsign/mgo/bson"
)

type Object struct {
	ID        bson.ObjectId `bson:"_id" json:"id"`
	CreatedAt time.Time     `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time     `bson:"updatedAt" json:"updatedAt"`
	Error     string        `bson:"error" json:"error"`
	ObjectDTO
}

type ObjectDTO struct {
	CurrentStep  fsm.VerticeName `bson:"currentStep" json:"currentStep"`
	CommandGraph string          `bson:"commandGraph" json:"commandGraph"`
}

type Storage interface {
	FindById(id bson.ObjectId) (Object, error)
	Find(selector bson.M) (Object, error)
	Create(obj ObjectDTO) (Object, error)
	Update(selector bson.M, update bson.M) error
}

type Repository struct {
}
