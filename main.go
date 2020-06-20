package main

import (
	"fmt"
	"github.com/karrick/godirwalk"
	"github.com/mpetavy/common"
	"sync"
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

	fileChannel   chan *FileMessage
	fileChannelWg sync.WaitGroup
	workerChannel chan struct{}
	workerWg      sync.WaitGroup
)

type FileMessage struct {
	path  string
	attrs *godirwalk.Dirent
}

type Metadata map[string]string

type DocumentRec struct {
	Path     string
	Metadata *Metadata
}

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

	fileChannel = make(chan *FileMessage, cfg.System.ChannelSize)
	fileChannelWg = sync.WaitGroup{}
	workerChannel = make(chan struct{}, cfg.System.WorkerSize)
	workerWg = sync.WaitGroup{}

	fileChannelWg.Add(1)

	go func() {
		common.Info("Channel started")
		defer func() {
			fileChannelWg.Done()
			common.Info("Channel stopped")
		}()

		for {
			var fileMessage *FileMessage
			var ok bool

			select {
			case fileMessage, ok = <-fileChannel:
				if !ok {
					return
				}
			case <-common.AppLifecycle().Channel():
				return
			}

			workerChannel <- struct{}{}
			workerWg.Add(1)

			go func(fileMessage *FileMessage) {
				defer func() {
					workerWg.Done()

					<-workerChannel
				}()

				metadata, err := cfg.Indexer.indexFile(fileMessage)
				if common.Error(err) {
					return
				}

				err = cfg.MongoDB.Save("doc", DocumentRec{
					Path:     fileMessage.path,
					Metadata: metadata,
				})
				if common.Error(err) {
					return
				}

				common.Debug("%v\n", metadata)
			}(fileMessage)
		}
	}()

	start := time.Now()

	err = cfg.Filesystem.InitialScan(func(path string, attrs *godirwalk.Dirent) error {
		fileChannel <- &FileMessage{path, attrs}

		return nil
	})

	close(fileChannel)

	fileChannelWg.Wait()

	common.Info("Time elapsed: %v", time.Since(start))

	if common.Error(err) {
		return err
	}

	return nil
}

func stop() error {
	workerWg.Wait()

	common.Error(cfg.Filesystem.Close())
	common.Error(cfg.Indexer.Close())
	common.Error(cfg.MongoDB.Close())

	return nil
}

func main() {
	defer common.Done()

	common.Run(nil)
}
