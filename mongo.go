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
	Hostname   string `json:"hostname" html:"Hostname"`
	Port       int    `json:"port" html:"Port"`
	SSL        bool   `json:"ssl" html:"SSL"`
	Database   string `json:"database" html:"Database"`
	Timeout    int    `json:"timeout" html:"Timeout"`
	PoolSize   int    `json:"poolSize" html:"Pool size"`
	Collection string `json:"collection" html:"Collection"`

	url  string
	pool chan *mongo.Client
}

type DocumentRec struct {
	Path     string
	Metadata *Metadata
}

var (
	_true bool = true
)

func NewMongo(mgo *MongoCfg) error {
	mgo.url = fmt.Sprintf("mongodb://%s:%d/?readPreference=primary&appname=%s&ssl=%v", mgo.Hostname, mgo.Port, common.Title(), mgo.SSL)
	if mgo.Timeout == 0 {
		mgo.Timeout = 3000
	}

	common.Info("MongoDB start: %v", mgo.url)

	mgo.pool = make(chan *mongo.Client, mgo.PoolSize)
	ce := common.ChannelError{}
	wg := sync.WaitGroup{}

	for i := 0; i < mgo.PoolSize; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			client, err := mongo.Connect(createCtx(mgo), options.Client().
				SetAppName(common.Title()).ApplyURI(mgo.url))
			if common.Error(err) {
				ce.Add(err)
			}

			err = client.Ping(nil, nil)
			if common.Error(err) {
				ce.Add(err)
			}

			mod := mongo.IndexModel{
				Keys: bson.M{
					"path": 1, // index in ascending order
				},
				Options: &options.IndexOptions{
					Background: &_true,
					Sparse:     &_true,
					Unique:     &_true,
				},
			}

			_, err = client.Database(mgo.Database).Collection(mgo.Collection).Indexes().CreateOne(context.Background(), mod)
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
	common.Info("MongoDB stop")

	close(mgo.pool)

	for client := range mgo.pool {
		if client != nil {

			common.Error(client.Disconnect(nil))
		}
	}

	return nil
}

func (mgo *MongoCfg) Insert(rec *DocumentRec) error {
	client := <-mgo.pool
	defer func() {
		mgo.pool <- client
	}()

	b, err := bson.Marshal(rec)
	if common.Error(err) {
		return err
	}

	_, err = client.Database(mgo.Database).
		Collection(mgo.Collection).
		InsertOne(context.Background(), b)

	return err
}

func (mgo *MongoCfg) Upsert(rec *DocumentRec) error {
	client := <-mgo.pool
	defer func() {
		mgo.pool <- client
	}()

	filter := bson.D{{"path", rec.Path}}

	_, err := client.
		Database(mgo.Database).
		Collection(mgo.Collection).
		UpdateOne(context.Background(), filter, bson.D{{"$set", rec}}, &options.UpdateOptions{Upsert: &_true})

	return err
}

func (mgo *MongoCfg) Delete(collectionName string, path string) error {
	client := <-mgo.pool
	defer func() {
		mgo.pool <- client
	}()

	_, err := client.
		Database(mgo.Database).
		Collection(collectionName).
		DeleteOne(context.Background(), bson.D{{"path", path}})

	return err
}
