package main

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/karrick/godirwalk"
	"github.com/mpetavy/common"
)

type ErrCannotIndex struct {
	path    string
	casedBy string
}

func (e *ErrCannotIndex) Error() string {
	return fmt.Sprintf("Cannot index path or file: %s Caused by: %s", e.path, e.casedBy)
}

type FilesystemCfg struct {
	Path              string   `json:"path" html:"Path"`
	Recursive         bool     `json:"recursive" html:"Recursive"`
	FileIncludes      []string `json:"fileIncludes" html:"File includes"`
	FileExcludes      []string `json:"fileExcludes" html:"´File excludes"`
	DirectoryIncludes []string `json:"directoryIncludes" html:"Directory includes"`
	DirectoryExcludes []string `json:"directoryExcludes" html:"Directory excludes"`

	watcher *fsnotify.Watcher
}

func NewFilesystem(fs *FilesystemCfg) error {
	b, err := common.FileExists(fs.Path)
	if common.Error(err) {
		return err
	}

	if !b {
		return fmt.Errorf("file or path not found: %s", fs.Path)
	}

	common.Info("Filesystem open: %v", fs.Path)

	return nil
}

func (fs *FilesystemCfg) InitialScan(walkFunc godirwalk.WalkFunc) error {
	var err error

	fs.watcher, err = fsnotify.NewWatcher()
	if common.Error(err) {
		return err
	}

	err = godirwalk.Walk(fs.Path, &godirwalk.Options{
		ErrorCallback: func(path string, err error) godirwalk.ErrorAction {
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

			if attrs.ModeType().IsDir() {
				common.Info("Add watcher: %v", path)

				return fs.watcher.Add(path)
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

func (fs *FilesystemCfg) Close() error {
	if fs.watcher != nil {
		common.Info("Watcher close")

		common.Error(fs.watcher.Close())
	}

	b, _ := common.FileExists(fs.Path)

	if b {
		common.Info("Filesystem close")
	}

	return nil
}
