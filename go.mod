module metafiler

go 1.16

require (
	github.com/fsnotify/fsnotify v1.4.9
	github.com/golang/snappy v0.0.4 // indirect
	github.com/karrick/godirwalk v1.16.1
	github.com/klauspost/compress v1.13.1 // indirect
	github.com/labstack/echo-contrib v0.11.0
	github.com/labstack/echo/v4 v4.9.0
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mpetavy/common v1.4.36
	github.com/mpetavy/go-dicom v0.0.0-20210302105037-44b79120da96
	github.com/quasoft/memstore v0.0.0-20191010062613-2bce066d2b0b
	github.com/youmark/pkcs8 v0.0.0-20201027041543-1326539a0a0a // indirect
	go.mongodb.org/mongo-driver v1.7.0
	golang.org/x/crypto v0.0.0-20220926161630-eccd6366d1be // indirect
	golang.org/x/net v0.0.0-20220930213112-107f3e3c3b0b // indirect
	golang.org/x/sys v0.0.0-20220928140112-f11e5e49a4ec // indirect
)

//replace github.com/mpetavy/common => ../common
