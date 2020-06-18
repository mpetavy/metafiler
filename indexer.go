package main

import (
	"fmt"
	"github.com/karrick/godirwalk"
	"github.com/mpetavy/common"
	"github.com/mpetavy/go-dicom"
	"github.com/mpetavy/go-dicom/dicomtag"
)

type IndexMessage struct {
	path  string
	attrs *godirwalk.Dirent
}

type IndexerCfg struct {
	TagIncludes []string `json:"dicomtagIncludes" html:"Dicomtag includes"`
	TagExcludes []string `json:"dicomtagExcludes" html:"Dicomtag excludes"`
	TagRenames  []string `json:"dicomtagRenames" html:"Dicomtag renames"`

	Channel chan *IndexMessage `json:"-"`
}

type Metadata map[string]string

func NewIndexer(indexer *IndexerCfg) error {
	common.Info("Indexer open")

	indexer.Channel = make(chan *IndexMessage, 100)

	go func() {
		common.Info("Index listener started")
		defer common.Info("Index listener stopped")

		for im := range indexer.Channel {
			common.Error(indexer.indexFile(im))
		}
	}()

	return nil
}

func (indexer *IndexerCfg) indexFile(im *IndexMessage) error {
	common.Info("%v", im)

	metadata, err := indexer.indexDicomFile(im.path)
	if common.Error(err) {
		return err
	}

	fmt.Printf("%v\n", metadata)

	return nil
}

func (indexer *IndexerCfg) indexDicomFile(path string) (Metadata, error) {
	metadata := make(Metadata)

	dataset, err := dicom.ReadDataSetFromFile(path, dicom.ReadOptions{
		DropPixelData: true,
		ReturnTags:    nil,
		StopAtTag:     nil,
	})

	if common.Error(err) {
		return nil, &ErrCannotIndex{
			path:    path,
			casedBy: err.Error(),
		}
	}

	for _, elem := range dataset.Elements {
		if elem.Tag != dicomtag.PixelData {
			v, err := elem.GetString()
			if err == nil {
				tn, err := dicomtag.FindTagInfo(elem.Tag)
				if err == nil {
					metadata[tn.Name] = v
				}
			}
		}
	}

	return metadata, nil
}

func (indexer *IndexerCfg) Close() error {
	common.Info("Indexer close")

	return nil
}
