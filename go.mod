module metafiler

go 1.16

require (
	github.com/fsnotify/fsnotify v1.4.9
	github.com/karrick/godirwalk v1.15.6
	github.com/labstack/echo-contrib v0.9.0
	github.com/labstack/echo/v4 v4.2.0
	github.com/mpetavy/common v1.2.15
	github.com/mpetavy/go-dicom v0.0.0-20200615105037-742a1dfb9324
	github.com/quasoft/memstore v0.0.0-20191010062613-2bce066d2b0b
	go.mongodb.org/mongo-driver v1.3.4
)

//replace github.com/mpetavy/common => ../common
