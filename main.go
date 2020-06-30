package main

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/karrick/godirwalk"
	"github.com/mpetavy/common"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	LDFLAG_DEVELOPER = "mpetavy"                                                    // will be replaced with ldflag
	LDFLAG_HOMEPAGE  = fmt.Sprintf("https://github.com/mpetavy/%s", common.Title()) // will be replaced with ldflag
	LDFLAG_LICENSE   = common.GPL2                                                  // will be replaced with ldflag
	LDFLAG_VERSION   = "1.0.0"                                                      // will be replaced with ldflag
	LDFLAG_EXPIRE    = "01.07.2020"                                                 // will be replaced with ldflag
	LDFLAG_GIT       = ""                                                           // will be replaced with ldflag
	LDFLAG_COUNTER   = "9999"                                                       // will be replaced with ldflag
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
	Path          string
	IsInitialScan bool
	IsCreated     bool
	IsWritten     bool
	IsDeleted     bool
	IsRenamed     bool
	IsChmoded     bool
}

type Metadata map[string]string

func init() {
	common.Init(true, LDFLAG_VERSION, "2020", "file system indexing", LDFLAG_DEVELOPER, LDFLAG_HOMEPAGE, LDFLAG_LICENSE, start, stop, nil, 0)

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

	return licenseDate.After(time.Now()), fmt.Errorf(common.Translate("This is an ALPHA software release. For internal usage/testing only. Expire date %v", licenseDate))
}

func formatMsg(registerMsg RegisterMsg) string {
	var sb strings.Builder

	sb.WriteString(registerMsg.Path)

	if registerMsg.IsInitialScan {
		sb.WriteString(" INITIALSCAN")
	}
	if registerMsg.IsCreated {
		sb.WriteString(" CREATED")
	}
	if registerMsg.IsWritten {
		sb.WriteString(" WRITTEN")
	}
	if registerMsg.IsDeleted {
		sb.WriteString(" DELETED")
	}
	if registerMsg.IsRenamed {
		sb.WriteString(" RENAMED")
	}
	if registerMsg.IsChmoded {
		sb.WriteString(" CHMODED")
	}

	return sb.String()
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

	workerCh = make(chan struct{}, cfg.Filesystem.CountWorkers)
	workerWg = sync.WaitGroup{}

	registerCh = make(chan *RegisterMsg, 10000000)
	registerWg = sync.WaitGroup{}
	registerWg.Add(1)

	common.Info("Registration start")

	startTime = time.Now()

	go func() {
		defer func() {
			common.Info("Registration stop")

			registerWg.Done()
		}()

		for {
			var registerMsg *RegisterMsg

			select {
			case registerMsg = <-registerCh:
				if registerMsg == nil {
					return
				}

				if registerMsg.Path == "" {
					common.Info("Initial scan stop: %v", time.Since(startTime))

					if !common.IsRunningAsService() {
						return
					}

					continue
				}

				if registerMsg.IsInitialScan {
					workerCh <- struct{}{}
					workerWg.Add(1)

					go func(registerMsg RegisterMsg) {
						defer func() {
							workerWg.Done()

							<-workerCh
						}()

						common.Error(processMsg(&registerMsg))
					}(*registerMsg)
				} else {
					common.Error(processMsg(registerMsg))
				}
			case event := <-cfg.Filesystem.watcher.Events:
				registerCh <- &RegisterMsg{
					Path:          event.Name,
					IsInitialScan: false,
					IsCreated:     event.Op&fsnotify.Create == fsnotify.Create,
					IsWritten:     event.Op&fsnotify.Write == fsnotify.Write,
					IsDeleted:     event.Op&fsnotify.Remove == fsnotify.Remove,
					IsRenamed:     event.Op&fsnotify.Rename == fsnotify.Rename,
					IsChmoded:     event.Op&fsnotify.Chmod == fsnotify.Chmod,
				}
			case err := <-cfg.Filesystem.watcher.Errors:
				common.Error(err)
			}
		}
	}()

	if !cfg.Filesystem.SkipInitialScan {
		common.Info("Initial scan start")

		err = cfg.Filesystem.InitialScan(func(path string, attrs *godirwalk.Dirent) error {
			registerCh <- &RegisterMsg{
				Path:          path,
				IsInitialScan: true,
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
			Path:          "",
			IsInitialScan: true,
			IsCreated:     false,
			IsWritten:     false,
			IsDeleted:     false,
			IsRenamed:     false,
			IsChmoded:     false,
		}

		if !common.IsRunningAsService() {
			registerWg.Wait()
			workerWg.Wait()
		}
	}

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
		if registerMsg.IsCreated || registerMsg.IsInitialScan {
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

	if registerMsg.IsWritten || registerMsg.IsInitialScan {
		return indexFile(registerMsg)
	}

	if registerMsg.IsDeleted {
		return removeFile(registerMsg)
	}

	return nil
}

func indexFile(registerMsg *RegisterMsg) error {
	if cfg.Filesystem.LogEvents {
		common.Info("Index file: %+v", formatMsg(*registerMsg))
	}

	metadata, err := cfg.Indexer.indexFile(registerMsg)
	if common.Error(err) {
		return err
	}

	err = cfg.MongoDB.Upsert(&DocumentRec{
		Path:     registerMsg.Path,
		Metadata: *metadata,
	})
	if common.Error(err) {
		return err
	}

	return nil
}

func removeFile(registerMsg *RegisterMsg) error {
	if cfg.Filesystem.LogEvents {
		common.Info("Remove file: %+v", formatMsg(*registerMsg))
	}

	err := cfg.MongoDB.Delete("doc", registerMsg.Path)
	if common.Error(err) {
		return err
	}

	return nil
}

func stop() error {
	if registerCh != nil {
		close(registerCh)
	}

	registerWg.Wait()
	workerWg.Wait()

	if cfg != nil {
		common.Error(cfg.Filesystem.Close())
		common.Error(cfg.Indexer.Close())
		common.Error(cfg.MongoDB.Close())
	}

	return nil
}

func main() {
	defer common.Done()

	common.Run(nil)
}
