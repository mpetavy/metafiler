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

type ErrFileNotFound struct {
	path string
}

func (e *ErrFileNotFound) Error() string {
	return fmt.Sprintf("File not found: %s", e.path)
}

type FilesystemCfg struct {
	Path              string   `json:"path" html:"Path"`
	Recursive         bool     `json:"recursive" html:"Recursive"`
	FileIncludes      []string `json:"fileIncludes" html:"File includes"`
	FileExcludes      []string `json:"fileExcludes" html:"´File excludes"`
	DirectoryIncludes []string `json:"directoryIncludes" html:"Directory includes"`
	DirectoryExcludes []string `json:"directoryExcludes" html:"Directory excludes"`

	Watcher *fsnotify.Watcher
	watches map[string]struct{}
}

func NewFilesystem(fs *FilesystemCfg) error {
	b, err := common.FileExists(fs.Path)
	if common.Error(err) {
		return err
	}

	if !b {
		return fmt.Errorf("file or path not found: %s", fs.Path)
	}

	common.Info("Filesystem Watcher start")

	fs.Watcher, err = fsnotify.NewWatcher()
	if common.Error(err) {
		return err
	}

	common.Info("Filesystem start: %v", fs.Path)

	return nil
}

func (fs *FilesystemCfg) InitialScan(walkFunc godirwalk.WalkFunc) error {
	err := godirwalk.Walk(fs.Path, &godirwalk.Options{
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

				return fs.Watcher.Add(path)
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
	common.Info("Add watcher: %v", path)

	err := fs.Watcher.Add(path)
	if common.Error(err) {
		return err
	}

	fs.watches[path] = struct{}{}

	return nil
}

func (fs *FilesystemCfg) RemoveWatcher(path string) error {
	common.Info("Remove watcher: %v", path)

	err := fs.Watcher.Remove(path)
	if common.Error(err) {
		return err
	}

	delete(fs.watches, path)

	return nil
}

func (fs *FilesystemCfg) IsWatched(path string) bool {
	_, ok := fs.watches[path]

	return ok
}

func (fs *FilesystemCfg) Close() error {
	if fs.Watcher != nil {
		common.Info("Filesystem Watcher stop")

		common.Error(fs.Watcher.Close())
	}

	b, _ := common.FileExists(fs.Path)

	if b {
		common.Info("Filesystem stop")
	}

	return nil
}
