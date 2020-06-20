package main

import (
	"context"
	"fmt"
	"github.com/mpetavy/common"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"sync"
	"time"
)

type MongoCfg struct {
	Hostname string `json:"hostname" html:"Hostname"`
	Port     int    `json:"port" html:"Port"`
	SSL      bool   `json:"ssl" html:"SSL"`
	Database string `json:"database" html:"Database"`
	Timeout  int    `json:"timeout" html:"Timeout"`
	PoolSize int    `json:"poolSize" html:"Pool size"`

	url  string
	pool chan *mongo.Client
}

func NewMongo(mgo *MongoCfg) error {
	mgo.url = fmt.Sprintf("mongodb://%s:%d/?readPreference=primary&appname=%s&ssl=%v", mgo.Hostname, mgo.Port, common.Title(), mgo.SSL)
	if mgo.Timeout == 0 {
		mgo.Timeout = 3000
	}

	common.Info("MongoDB open: %v", mgo.url)

	mgo.pool = make(chan *mongo.Client, mgo.PoolSize)
	ce := common.ChannelError{}
	wg := sync.WaitGroup{}

	for i := 0; i < mgo.PoolSize; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			client, err := mongo.Connect(createCtx(mgo), options.Client().
				SetAppName(common.Title()).SetMaxPoolSize(100).ApplyURI(mgo.url))
			if common.Error(err) {
				ce.Add(err)
			}

			err = client.Ping(nil, nil)
			if common.Error(err) {
				ce.Add(err)
			}

			mgo.pool <- client
		}()
	}

	wg.Wait()

	if ce.Exists() {
		return ce.Get()
	}

	return nil
}

func createCtx(mgo *MongoCfg) context.Context {
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(common.Max(1000, mgo.Timeout))*time.Millisecond)

	return ctx
}

func (mgo *MongoCfg) Close() error {
	common.Info("MongoDB close")

	close(mgo.pool)

	for client := range mgo.pool {
		if client != nil {

			client.Disconnect(nil)
		}
	}

	return nil
}

func (mgo *MongoCfg) Save(collectionName string, v interface{}) error {
	client := <-mgo.pool
	defer func() {
		mgo.pool <- client
	}()

	b, err := bson.Marshal(v)
	if common.Error(err) {
		return err
	}

	_, err = client.Database(mgo.Database).Collection(collectionName).InsertOne(context.Background(), b)

	return err
}
