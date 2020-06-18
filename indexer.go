package main

import (
	"github.com/mpetavy/common"
	"github.com/mpetavy/go-dicom"
	"github.com/mpetavy/go-dicom/dicomtag"
)

type IndexerCfg struct {
	TagIncludes []string `json:"dicomtagIncludes" html:"Dicomtag includes"`
	TagExcludes []string `json:"dicomtagExcludes" html:"Dicomtag excludes"`
	TagRenames  []string `json:"dicomtagRenames" html:"Dicomtag renames"`
}

type Metadata map[string]string

func NewIndexer(indexer *IndexerCfg) error {
	common.Info("Indexer open")

	return nil
}

func (indexer *IndexerCfg) indexFile(fileMessage *FileMessage) (*Metadata, error) {
	//common.Info("Indexer file: %v", fileMessage.path)

	metadata, err := indexer.indexDicomFile(fileMessage.path)
	if common.Error(err) {
		return nil, err
	}

	return &metadata, nil
}

func (indexer *IndexerCfg) indexDicomFile(path string) (Metadata, error) {
	metadata := make(Metadata)

	dataset, err := dicom.ReadDataSetFromFile(path, dicom.ReadOptions{
		DropPixelData: true,
		ReturnTags:    nil,
		StopAtTag:     nil,
	})

	if err != nil {
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
