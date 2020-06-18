module metafiler

go 1.14

require (
	github.com/fsnotify/fsnotify v1.4.9
	github.com/karrick/godirwalk v1.15.6
	github.com/mpetavy/common v1.1.39
	github.com/mpetavy/go-dicom v0.0.0-20200615105037-742a1dfb9324
	go.mongodb.org/mongo-driver v1.3.4
)

replace github.com/mpetavy/common => ../common
