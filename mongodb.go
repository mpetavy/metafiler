package main

import (
	"context"
	"fmt"
	"github.com/mpetavy/common"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type MongoCfg struct {
	Hostname string `json:"hostname" html:"Hostname"`
	Port     int    `json:"port" html:"Port"`
	SSL      bool   `json:"ssl" html:"SSL"`
	Database string `json:"database" html:"Database"`
	Timeout  int    `json:"timeout" html:"Timeout"`

	url    string
	client *mongo.Client
}

func NewMongoDB(mongodb *MongoCfg) error {
	mongodb.url = fmt.Sprintf("mongodb://%s:%d/?readPreference=primary&appname=%s&ssl=%v", mongodb.Hostname, mongodb.Port, common.Title(), mongodb.SSL)
	if mongodb.Timeout == 0 {
		mongodb.Timeout = 3000
	}

	common.Info("MongoDB open: %v", mongodb.url)

	var err error

	mongodb.client, err = mongo.Connect(createCtx(mongodb), options.Client().
		SetAppName(common.Title()).SetMaxPoolSize(100).ApplyURI(mongodb.url))
	if common.Error(err) {
		return err
	}

	err = mongodb.client.Ping(nil, nil)
	if common.Error(err) {
		return err
	}

	return nil
}

func createCtx(mongodb *MongoCfg) context.Context {
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(common.Max(1000, mongodb.Timeout))*time.Millisecond)

	return ctx
}

func (mongodb *MongoCfg) Close() error {
	if mongodb.client != nil {
		common.Info("MongoDB close")

		return mongodb.client.Disconnect(nil)
	}

	return nil
}

func (mongodb *MongoCfg) Save(collectionName string, v interface{}) error {
	b, err := bson.Marshal(v)
	if common.Error(err) {
		return err
	}

	_, err = mongodb.client.Database(mongodb.Database).Collection(collectionName).InsertOne(context.Background(), b)

	return err
}
