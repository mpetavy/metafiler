package main

import (
	"fmt"
	"github.com/mpetavy/common"
)

type FilesystemCfg struct {
	Path              string   `json:"path" html:"Path"`
	Recursive         bool     `json:"recursive" html:"Recursive"`
	FileIncludes      []string `json:"fileIncludes" html:"File includes"`
	FileExcludes      []string `json:"fileExIncludes" html:"´File excludes"`
	DirectoryIncludes []string `json:"directoryIncludes" html:"Directory includes"`
	DirectoryExcludes []string `json:"directoryExcludes" html:"Directory excludes"`
}

func NewFilesystem(fs *FilesystemCfg) error {
	b, err := common.FileExists(fs.Path)
	if common.Error(err) {
		return err
	}

	if !b {
		return fmt.Errorf("file or path not found: %s", fs.Path)
	}

	common.Info("Open: %v", fs.Path)

	return nil
}

func (fs *FilesystemCfg) Scan() error {
	err := common.WalkFilepath(fs.Path, fs.Recursive, func(path string) error {
		return nil
	})

	if common.Error(err) {
		return err
	}

	return nil
}

func (fs *FilesystemCfg) Close() error {
	b, _ := common.FileExists(fs.Path)

	if b {
		common.Info("Close")
	}

	return nil
}
