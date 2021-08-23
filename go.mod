module metafiler

go 1.16

require (
	github.com/fsnotify/fsnotify v1.4.9
	github.com/golang/snappy v0.0.4 // indirect
	github.com/karrick/godirwalk v1.16.1
	github.com/klauspost/compress v1.13.1 // indirect
	github.com/labstack/echo-contrib v0.11.0
	github.com/labstack/echo/v4 v4.4.0
	github.com/mpetavy/common v1.3.8
	github.com/mpetavy/go-dicom v0.0.0-20210302105037-44b79120da96
	github.com/quasoft/memstore v0.0.0-20191010062613-2bce066d2b0b
	github.com/youmark/pkcs8 v0.0.0-20201027041543-1326539a0a0a // indirect
	go.mongodb.org/mongo-driver v1.7.0
)

//replace github.com/mpetavy/common => ../common
