package main

import (
	"context"
	"fmt"
	"github.com/mpetavy/common"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"sync"
)

type MongoCfg struct {
	Hostname     string `json:"hostname" html:"Hostname"`
	Port         int    `json:"port" html:"Port"`
	SSL          bool   `json:"ssl" html:"SSL"`
	Database     string `json:"database" html:"Database"`
	DropDatabase bool   `json:"dropDatabase" html:"Drop database"`
	Timeout      int    `json:"timeout" html:"Timeout"`
	CountHandles int    `json:"countHandles" html:"Count handles"`
	Collection   string `json:"collection" html:"Collection"`

	url  string
	pool chan *mongo.Client
}

type DocumentRec struct {
	Path     string
	Metadata Metadata
}

var (
	_true bool = true
)

func NewMongo(mgo *MongoCfg) error {
	mgo.url = fmt.Sprintf("mongodb://%s:%d/?readPreference=primary&appname=%s&ssl=%v", mgo.Hostname, mgo.Port, common.Title(), mgo.SSL)

	common.Info("MongoDB start: %v", mgo.url)

	mgo.pool = make(chan *mongo.Client, mgo.CountHandles)
	channelErrors := common.ChannelError{}
	wg := sync.WaitGroup{}

	for i := 0; i < mgo.CountHandles; i++ {
		wg.Add(1)

		go func() {
			defer common.UnregisterGoRoutine(common.RegisterGoRoutine(1))

			defer wg.Done()

			ctx, cancel := createCtx(mgo)
			defer cancel()
			client, err := mongo.Connect(ctx, options.Client().
				SetAppName(common.Title()).ApplyURI(mgo.url))
			if common.Error(err) {
				channelErrors.Add(err)
			}

			ctx, cancel = createCtx(mgo)
			defer cancel()
			err = client.Ping(ctx, nil)
			if common.Error(err) {
				channelErrors.Add(err)
			}

			mgo.pool <- client
		}()
	}

	wg.Wait()

	if channelErrors.Exists() {
		return channelErrors.Get()
	}

	if mgo.DropDatabase {
		err := mgo.Drop()
		if common.Error(err) {
			return err
		}

		err = mgo.CreateIndex()
		if common.Error(err) {
			return err
		}
	}

	return nil
}

func createCtx(mgo *MongoCfg) (context.Context, context.CancelFunc) {
	if mgo.Timeout == 0 {
		return context.Background(), func() {}
	}

	ctx, cancel := context.WithTimeout(context.Background(), common.MillisecondToDuration(mgo.Timeout))

	return ctx, cancel
}

func (mgo *MongoCfg) Close() error {
	common.Info("MongoDB stop")

	close(mgo.pool)

	for client := range mgo.pool {
		if client != nil {
			ctx, cancel := createCtx(mgo)
			defer cancel()
			common.Error(client.Disconnect(ctx))
		}
	}

	return nil
}

func (mgo *MongoCfg) Upsert(rec *DocumentRec) error {
	client := <-mgo.pool
	defer func() {
		mgo.pool <- client
	}()

	filter := bson.D{{"path", rec.Path}}

	ctx, cancel := createCtx(mgo)
	defer cancel()

	_, err := client.
		Database(mgo.Database).
		Collection(mgo.Collection).
		UpdateOne(ctx, filter, bson.D{{"$set", rec}}, &options.UpdateOptions{Upsert: &_true})

	return err
}

func (mgo *MongoCfg) Delete(collectionName string, path string) error {
	client := <-mgo.pool
	defer func() {
		mgo.pool <- client
	}()

	ctx, cancel := createCtx(mgo)
	defer cancel()

	_, err := client.
		Database(mgo.Database).
		Collection(collectionName).
		DeleteOne(ctx, bson.D{{"path", path}})

	return err
}

func (mgo *MongoCfg) Drop() error {
	common.Info("Drop database %s", mgo.Collection)

	client := <-mgo.pool
	defer func() {
		mgo.pool <- client
	}()

	ctx, cancel := createCtx(mgo)
	defer cancel()

	return client.Database(mgo.Database).Drop(ctx)
}

func (mgo *MongoCfg) CreateIndex() error {
	common.Info("Create index %s", mgo.Collection)

	client := <-mgo.pool
	defer func() {
		mgo.pool <- client
	}()

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

	ctx, cancel := createCtx(mgo)
	defer cancel()

	_, err := client.Database(mgo.Database).Collection(mgo.Collection).Indexes().CreateOne(ctx, mod)

	return err
}
