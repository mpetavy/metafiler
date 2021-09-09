package main

import (
	"encoding/json"
	"github.com/mpetavy/common"
	"runtime"
)

type Cfg struct {
	common.Configuration
	MongoDB    MongoCfg      `json:"mongodb" html:"Mongo DB"`
	Filesystem FilesystemCfg `json:"filesystem" html:"Filesystem"`
	Indexer    IndexerCfg    `json:"indexer" html:"Indexer"`
}

func NewCfg() (*Cfg, error) {
	cfg := &Cfg{}

	cfg.Filesystem.CountWorkers = runtime.NumCPU() * 2

	cfg.MongoDB.Hostname = "localhost"
	cfg.MongoDB.Port = 27017
	cfg.MongoDB.CountHandles = runtime.NumCPU()
	cfg.MongoDB.Collection = "doc"
	cfg.MongoDB.Timeout = 3000

	ba, err := common.LoadConfigurationFile()
	if common.Error(err) {
		return nil, err
	}

	if ba == nil {
		err := common.SaveConfiguration(cfg)
		if common.Error(err) {
			return nil, err
		}

		common.Info("Default configuration file generated")

		return nil, &common.ErrExit{}
	}

	err = json.Unmarshal(ba, cfg)
	if common.Error(err) {
		return nil, err
	}

	return cfg, nil
}
