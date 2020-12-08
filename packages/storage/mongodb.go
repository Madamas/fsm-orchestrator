package storage

import (
	"github.com/Madamas/fsm-orchestrator/packages/config"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
	"sync"
	"time"
)

// connection struct is barebone implementation that contains all that you need
// for simple work with mongodb
type connection struct {
	Config         config.MongodbConfig
	db             *mgo.Database
	session        *mgo.Session
	isConnected    bool
	isReconnecting bool
	mux            sync.Mutex
	err            error
}

func (c *connection) GetCollection(name string) (*mgo.Collection, error) {
	db, err := c.Db()

	if err != nil {
		return nil, err
	}

	return db.C(name), err
}

func (c *connection) Db() (*mgo.Database, error) {
	if c.isReconnecting {
		c.mux.Lock()
		defer c.mux.Unlock()
		return c.db, c.err
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	if err := c.ping(); err != nil {
		if err := c.reconnect(); err != nil {
			return nil, err
		}
	}

	return c.db, nil
}

func (c *connection) reconnect() error {
	if c.session != nil {
		go closeSession(c.session)
	}
	c.session = nil
	c.db = nil
	c.err = nil
	c.isConnected = false
	c.isReconnecting = true

	session, err := reconnect(c.Config)

	if err != nil {
		c.err = err
		c.isConnected = false
		c.isReconnecting = false
		return err
	}

	c.session = session
	c.db = session.DB(c.Config.Database)
	c.err = nil
	c.isConnected = true
	c.isReconnecting = false

	return nil
}

func (c *connection) ping() error {
	if c.session == nil {
		return errors.New("no active session")
	}
	return errors.Wrap(c.session.Ping(), "ping error")
}

func closeSession(session *mgo.Session) {
	time.Sleep(time.Minute * 10)
	session.Close()
}

func connect(cfg config.MongodbConfig) (*mgo.Session, error) {
	info, err := mgo.ParseURL(cfg.Url)
	if err != nil {
		return nil, err
	}
	info.Timeout = 200 * time.Millisecond
	info.ReadTimeout = 200 * time.Millisecond
	info.WriteTimeout = 200 * time.Millisecond

	return mgo.DialWithInfo(info)
}

func reconnect(cfg config.MongodbConfig) (*mgo.Session, error) {
	for i := 0; ; i++ {
		session, err := connect(cfg)

		if err == nil {
			return session, nil
		}

		time.Sleep(time.Second)
	}
}

func Connect(config config.MongodbConfig) (*connection, error) {
	session, err := connect(config)

	if err != nil {
		return nil, err
	}

	return &connection{
		Config:  config,
		session: session,
		db:      session.DB(config.Database),
	}, nil
}

func NewMongoStorage(config config.MongodbConfig) (*Repository, error) {
	conn, err := Connect(config)

	if err != nil {
		return nil, err
	}

	return NewRepository(&MongoStorage{
		conn: conn,
		name: config.Table,
	}), nil
}

type MongoStorage struct {
	conn *connection
	name string
}

func parseID(id string) (bson.ObjectId, error) {
	if !bson.IsObjectIdHex(id) {
		return "", errors.New("invalid ID")
	}
	return bson.ObjectIdHex(id), nil
}

func (ms *MongoStorage) Create(obj ObjectDTO) (*Object, error) {
	collection, err := ms.conn.GetCollection(ms.name)

	if err != nil {
		return nil, err
	}

	job := &Object{
		ID:           bson.NewObjectId(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Status:       obj.Status,
		CommandGraph: obj.CommandGraph,
		Params:       obj.Params,
	}
	if err = collection.Insert(job); err != nil {
		return nil, err
	}

	job.ID = job.ID.(bson.ObjectId).Hex()

	return job, nil
}

func (ms *MongoStorage) FindById(id string) (*Object, error) {
	bsonId, err := parseID(id)

	if err != nil {
		return nil, err
	}

	collection, err := ms.conn.GetCollection(ms.name)

	if err != nil {
		return nil, err
	}

	obj := new(Object)

	if err := collection.FindId(bsonId).One(obj); err != nil {
		return nil, err
	}

	obj.ID = obj.ID.(bson.ObjectId).Hex()

	return obj, nil
}

func operationMapper(operation OperationKey) string {
	switch operation {
	case AddOperation:
		return "$push"
	default:
		return string(operation)
	}
}

func (ms *MongoStorage) UpdateById(id string, update KV, operation OperationMap) error {
	bsonId, err := parseID(id)

	if err != nil {
		return err
	}

	collection, err := ms.conn.GetCollection(ms.name)

	if err != nil {
		return err
	}

	if update == nil {
		update = KV{}
	}
	update["updatedAt"] = time.Now()

	change := bson.M{
		"$set": update,
	}

	if operation != nil {
		for key, val := range operation {
			change[operationMapper(key)] = val
		}
	}

	if err := collection.UpdateId(bsonId, change); err != nil {
		return err
	}
	return nil
}
