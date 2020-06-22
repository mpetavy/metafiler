package main

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/karrick/godirwalk"
	"github.com/mpetavy/common"
	"os"
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
	cfg        *Cfg
	registerCh chan *RegisterMsg
	registerWg sync.WaitGroup
	workerCh   chan struct{}
	workerWg   sync.WaitGroup
	startTime  time.Time
)

type RegisterMsg struct {
	IsInitialScan bool
	Path          string
	IsCreated     bool
	IsWritten     bool
	IsDeleted     bool
	IsRenamed     bool
	IsChmoded     bool
}

type Metadata map[string]string

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

	err = NewMongo(&cfg.MongoDB)
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

	workerCh = make(chan struct{}, cfg.System.WorkerSize)
	workerWg = sync.WaitGroup{}

	registerCh = make(chan *RegisterMsg, cfg.System.ChannelSize)
	registerWg = sync.WaitGroup{}
	registerWg.Add(1)

	common.Info("Registration start")

	go func() {
		defer func() {
			registerWg.Done()

			common.Info("Registration finished")
		}()

		for {
			var registerMsg *RegisterMsg

			select {
			case registerMsg = <-registerCh:
				if registerMsg.Path == "" {
					common.Info("Initial scan finished")

					if !common.IsRunningAsService() {
						return
					}

					continue
				}

				if registerMsg.IsInitialScan {
					workerCh <- struct{}{}
					workerWg.Add(1)

					go func(registerMsg *RegisterMsg) {
						defer func() {
							workerWg.Done()

							<-workerCh
						}()

						common.Error(indexFile(registerMsg))
					}(registerMsg)
				} else {
					common.Info("Filesystem event: %+v", registerMsg)

					common.Error(processMsg(registerMsg))
				}
			case event := <-cfg.Filesystem.Watcher.Events:
				registerCh <- &RegisterMsg{
					IsInitialScan: false,
					Path:          event.Name,
					IsCreated:     event.Op&fsnotify.Create == fsnotify.Create,
					IsWritten:     event.Op&fsnotify.Write == fsnotify.Write,
					IsDeleted:     event.Op&fsnotify.Remove == fsnotify.Remove,
					IsRenamed:     event.Op&fsnotify.Rename == fsnotify.Rename,
					IsChmoded:     event.Op&fsnotify.Chmod == fsnotify.Chmod,
				}
			case err := <-cfg.Filesystem.Watcher.Errors:
				common.Error(err)
			case <-common.AppLifecycle().Channel():
				return
			}
		}
	}()

	startTime = time.Now()

	common.Info("Initial scan start")

	err = cfg.Filesystem.InitialScan(func(path string, attrs *godirwalk.Dirent) error {
		registerCh <- &RegisterMsg{
			IsInitialScan: true,
			Path:          path,
			IsCreated:     false,
			IsWritten:     false,
			IsDeleted:     false,
			IsRenamed:     false,
			IsChmoded:     false,
		}

		return nil
	})
	if common.Error(err) {
		return err
	}

	registerCh <- &RegisterMsg{
		IsInitialScan: true,
		Path:          "",
		IsCreated:     false,
		IsWritten:     false,
		IsDeleted:     false,
		IsRenamed:     false,
		IsChmoded:     false,
	}

	registerWg.Wait()

	return nil
}

func processMsg(registerMsg *RegisterMsg) error {
	var fi os.FileInfo

	isDir := false

	if cfg.Filesystem.IsWatched(registerMsg.Path) {
		isDir = true
	} else {
		fi, _ = os.Stat(registerMsg.Path)

		isDir = fi != nil && fi.IsDir()
	}

	if isDir {
		if registerMsg.IsCreated {
			err := cfg.Filesystem.AddWatcher(registerMsg.Path)
			if common.Error(err) {
				return err
			}
		}

		if registerMsg.IsDeleted {
			err := cfg.Filesystem.RemoveWatcher(registerMsg.Path)
			if common.Error(err) {
				return err
			}
		}

		return nil
	}

	if registerMsg.IsCreated {
		return indexFile(registerMsg)
	}

	if registerMsg.IsWritten {
		err := removeFile(registerMsg)
		if common.Error(err) {
			return err
		}

		return indexFile(registerMsg)
	}

	if registerMsg.IsDeleted {
		return removeFile(registerMsg)
	}

	return nil
}

func indexFile(registerMsg *RegisterMsg) error {
	metadata, err := cfg.Indexer.indexFile(registerMsg)
	if common.Error(err) {
		return err
	}

	err = cfg.MongoDB.Upsert(&DocumentRec{
		Path:     registerMsg.Path,
		Metadata: metadata,
	})
	if common.Error(err) {
		return err
	}

	return nil
}

func removeFile(registerMsg *RegisterMsg) error {
	err := cfg.MongoDB.Delete("doc", registerMsg.Path)
	if common.Error(err) {
		return err
	}

	return nil
}

func stop() error {
	common.Info("Time elapsed: %v", time.Since(startTime))

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
