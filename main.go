package main

import (
	"fmt"
	"github.com/karrick/godirwalk"
	"github.com/mpetavy/common"
	"time"
)

var (
	LDFLAG_VERSION = "1.0.0"      // will be replaced with ldflag
	LDFLAG_EXPIRE  = "01.07.2020" // will be replaced with ldflag
	LDFLAG_GIT     = ""           // will be replaced with ldflag
	LDFLAG_COUNTER = "9999"       // will be replaced with ldflag
)

var (
	cfg *Cfg
)

func init() {
	common.Init(true, LDFLAG_VERSION, "2020", "observes directory paths and index metadata", "mpetavy", fmt.Sprintf("https://github.com/mpetavy/%s", common.Title()), common.APACHE, start, stop, nil, 0)

	common.Events.NewFuncReceiver(common.EventFlagsSet{}, func(ev common.Event) {
		common.Debug("LDFLAG_VERSION: %s\n", LDFLAG_VERSION)
		common.Debug("LDFLAG_EXPIRE: %s\n", LDFLAG_EXPIRE)
		common.Debug("LDFLAG_GIT: %s\n", LDFLAG_GIT)
		common.Debug("LDFLAG_COUNTER: %s\n", LDFLAG_COUNTER)
	})

	var err error

	ok, err := CheckLicense()
	if !ok {
		common.Error(err)

		common.Exit(1)
	} else {
		if err != nil {
			common.Warn(err.Error())
		}
	}
}

func CheckLicense() (bool, error) {
	if LDFLAG_EXPIRE == "" {
		return true, nil
	}

	licenseDate, err := common.ParseDateTime(common.DateMask, LDFLAG_EXPIRE)
	if common.Error(err) {
		return false, err
	}

	return licenseDate.After(time.Now()), fmt.Errorf(common.Translate("This is an ALPHA software release. For ZEISS internal usage/testing only. Expire date %v", licenseDate))
}

func start() error {
	var err error

	cfg, err = NewCfg()
	if common.Error(err) {
		return err
	}

	err = NewMongoDB(&cfg.MongoDB)
	if common.Error(err) {
		return err
	}

	err = NewIndexer(&cfg.Indexer)
	if common.Error(err) {
		return err
	}

	err = NewFilesystem(&cfg.Filesystem)
	if common.Error(err) {
		return err
	}

	start := time.Now()
	err = cfg.Filesystem.InitialScan(func(path string, attrs *godirwalk.Dirent) error {
		if common.AppLifecycle().IsSet() {
			cfg.Indexer.Channel <- &IndexMessage{path, attrs}
		}

		return nil
	})

	if common.Error(err) {
		return err
	}
	fmt.Printf("time elapsed: %v\n", time.Since(start))

	return nil
}

func stop() error {
	common.Error(cfg.Filesystem.Close())
	common.Error(cfg.Indexer.Close())
	common.Error(cfg.MongoDB.Close())

	return nil
}

func main() {
	defer common.Done()

	common.Run(nil)
}
