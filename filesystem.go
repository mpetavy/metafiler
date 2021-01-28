package main

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/karrick/godirwalk"
	"github.com/mpetavy/common"
	"sync"
)

type ErrCannotIndex struct {
	path    string
	casedBy string
}

func (e *ErrCannotIndex) Error() string {
	return fmt.Sprintf("Cannot index path or file: %s Caused by: %s", e.path, e.casedBy)
}

type ErrSkipNode struct {
	path string
}

func (e *ErrSkipNode) Error() string {
	return fmt.Sprintf("SkipNode: %s", e.path)
}

type FilesystemCfg struct {
	Path              string   `json:"path" html:"Path"`
	Recursive         bool     `json:"recursive" html:"Recursive"`
	FileIncludes      []string `json:"fileIncludes" html:"File includes"`
	FileExcludes      []string `json:"fileExcludes" html:"´File excludes"`
	DirectoryIncludes []string `json:"directoryIncludes" html:"Directory includes"`
	DirectoryExcludes []string `json:"directoryExcludes" html:"Directory excludes"`
	CountWorkers      int      `json:"countWorkers" html:"Count workers"`
	LogEvents         bool     `json:"logEvents" html:"Log events"`
	SkipInitialScan   bool     `json:"skipInitialScan" html:"Skip initial scan"`

	watcher *fsnotify.Watcher
	watches map[string]struct{}
	mu      sync.Mutex
}

func NewFilesystem(fs *FilesystemCfg) error {
	fs.Path = common.CleanPath(fs.Path)

	if !common.FileExists(fs.Path) {
		return &common.ErrFileNotFound{fs.Path}
	}

	common.Info("Filesystem start: %v", fs.Path)

	common.Info("Filesystem Watcher start")

	var err error

	fs.watcher, err = fsnotify.NewWatcher()
	if common.Error(err) {
		return err
	}

	fs.watches = make(map[string]struct{})

	return nil
}

func (fs *FilesystemCfg) InitialScan(walkFunc godirwalk.WalkFunc) error {
	err := godirwalk.Walk(fs.Path, &godirwalk.Options{
		ErrorCallback: func(path string, err error) godirwalk.ErrorAction {
			if _, ok := err.(*ErrSkipNode); ok {
				return godirwalk.SkipNode
			}

			if _, ok := err.(*common.ErrExit); ok {
				return godirwalk.Halt
			}

			common.Error(&ErrCannotIndex{
				path:    path,
				casedBy: err.Error(),
			})

			return godirwalk.SkipNode
		},
		FollowSymbolicLinks: false,
		Unsorted:            true,
		Callback: func(path string, attrs *godirwalk.Dirent) error {
			if !common.AppLifecycle().IsSet() {
				return &common.ErrExit{}
			}

			if !fs.Recursive && attrs.IsDir() && path != fs.Path {
				return &ErrSkipNode{}
			}

			return walkFunc(path, attrs)
		},
		PostChildrenCallback: nil,
		ScratchBuffer:        nil,
		AllowNonDirectory:    false,
	})
	if common.Error(err) {
		return err
	}

	return nil
}

func (fs *FilesystemCfg) AddWatcher(path string) error {
	if !fs.Recursive && path != fs.Path {
		return nil
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.LogEvents {
		common.Info("Add watcher: %v", path)
	}

	fs.watches[path] = struct{}{}

	err := fs.watcher.Add(path)
	if common.Error(err) {
		return err
	}

	return nil
}

func (fs *FilesystemCfg) RemoveWatcher(path string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.LogEvents {
		common.Info("Remove watcher: %v", path)
	}

	delete(fs.watches, path)

	// bug in fsnotify: sometimes the file is already physical deleted at first and then watcher.Remove breaks
	common.DebugError(fs.watcher.Remove(path))

	return nil
}

func (fs *FilesystemCfg) IsWatched(path string) bool {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	_, ok := fs.watches[path]

	return ok
}

func (fs *FilesystemCfg) Close() error {
	if fs.watcher != nil {
		common.Info("Filesystem Watcher stop")

		common.Error(fs.watcher.Close())
	}

	if common.FileExists(fs.Path) {
		common.Info("Filesystem stop")
	}

	return nil
}
