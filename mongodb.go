package main

import (
	"context"
	"fmt"
	"github.com/mpetavy/common"
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

	URL    string        `json:"-"`
	Client *mongo.Client `json:"-"`
}

func NewMongoDB(mongodb *MongoCfg) error {
	mongodb.URL = fmt.Sprintf("mongodb://%s:%d/?readPreference=primary&appname=%s&ssl=%v", mongodb.Hostname, mongodb.Port, common.Title(), mongodb.SSL)
	if mongodb.Timeout == 0 {
		mongodb.Timeout = 3000
	}

	common.Info("MongoDB open: %v", mongodb.URL)

	var err error

	mongodb.Client, err = mongo.Connect(createCtx(mongodb), options.Client().
		SetAppName(common.Title()).ApplyURI(mongodb.URL))
	if common.Error(err) {
		return err
	}

	err = mongodb.Client.Ping(nil, nil)
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
	if mongodb.Client != nil {
		common.Info("MongoDB close: %v", mongodb.URL)

		return mongodb.Client.Disconnect(nil)
	}

	return nil
}
