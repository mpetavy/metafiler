package main

import (
	"encoding/json"
	"github.com/mpetavy/common"
	"io/ioutil"
)

type SystemCfg struct {
	ChannelSize int `json:"channelSize" html:"Channel size"`
	WorkerSize  int `json:"workerSize" html:"Worker Size"`
}
type Cfg struct {
	common.Configuration
	System     SystemCfg     `json:"system" html:"System"`
	MongoDB    MongoCfg      `json:"mongodb" html:"Mongo DB"`
	Filesystem FilesystemCfg `json:"filesystem" html:"Filesystem"`
	Indexer    IndexerCfg    `json:"indexer" html:"Indexer"`
}

func NewCfg() (*Cfg, error) {
	cfg := &Cfg{}

	cfg.Flags = make(map[string]string)

	ba := common.GetConfigurationBuffer()
	if ba == nil {
		var err error

		ba, err = json.MarshalIndent(cfg, "", "    ")
		if common.Error(err) {
			return nil, err
		}

		fn := common.AppFilename(".json")

		err = ioutil.WriteFile(fn, ba, common.DefaultFileMode)
		if common.Error(err) {
			return nil, err
		}

		common.Info("Default configuration file %v generated", fn)

		return nil, &common.ErrExit{}
	}

	err := json.Unmarshal(ba, cfg)
	if common.Error(err) {
		return nil, err
	}

	return cfg, nil
}
