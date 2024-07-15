package main

import (
	"github.com/mpetavy/common"
	"runtime"
)

type MetafilerCfg struct {
	common.Configuration
	MongoDB    MongoCfg      `json:"mongodb" html:"Mongo DB"`
	Filesystem FilesystemCfg `json:"filesystem" html:"Filesystem"`
	Indexer    IndexerCfg    `json:"indexer" html:"Indexer"`
}

func NewCfg() (*MetafilerCfg, error) {
	cfg, err := common.LoadConfigurationFile[MetafilerCfg]()
	if common.Error(err) {
		return nil, err
	}

	if cfg != nil {
		return cfg, nil
	}

	cfg = &MetafilerCfg{}

	cfg.Filesystem.CountWorkers = runtime.NumCPU() * 2

	cfg.MongoDB.Hostname = "localhost"
	cfg.MongoDB.Port = 27017
	cfg.MongoDB.CountHandles = runtime.NumCPU()
	cfg.MongoDB.Collection = "doc"
	cfg.MongoDB.Timeout = 3000

	err = common.SaveConfigurationFile(cfg)
	if common.Error(err) {
		return nil, err
	}

	common.Info("Default configuration file generated")

	return nil, &common.ErrExit{}
}
