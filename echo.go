package main

import (
	"crypto/tls"
	"fmt"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mpetavy/common"
	"github.com/quasoft/memstore"
	"net"
	"net/http"
	"time"
)

const (
	ROUTE_HOME   = "/"
	ROUTE_LOGOUT = "/logout"
	ROUTE_STATIC = "/static/*"
)

type EchoCfg struct {
	Port int  `json:"port" html:"Port"`
	Tls  bool `json:"bool" html:"TLS"`

	ecco       *echo.Echo
	store      *memstore.MemStore
	httpServer *http.Server
}

func NewEcho(ecco *EchoCfg) error {
	common.Info("Web start")

	ecco.ecco = echo.New()

	loggerConfig := middleware.DefaultLoggerConfig
	loggerConfig.Output = common.NewEchoLogger()

	secret, err := common.RndBytes(32)
	if common.Error(err) {
		return err
	}

	ecco.store = memstore.NewMemStore(secret)
	ecco.httpServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", ecco.Port),
		ReadTimeout:       time.Minute,
		ReadHeaderTimeout: time.Minute,
		WriteTimeout:      time.Minute,
		ConnState: func(conn net.Conn, cs http.ConnState) {
			switch cs {
			case http.StateNew:
				common.Error(conn.SetReadDeadline(time.Now().Add(time.Minute)))
			case http.StateActive:
				tlsConn, ok := conn.(*tls.Conn)
				if ok {
					common.TlsDebugConnection(fmt.Sprintf("HTTP :%d", ecco.Port), tlsConn)
				}
			default:
				// NOTE: this is a good place to track connection level metrics :)
			}
		},
	}

	if ecco.Tls {
		tlsConfig, err := common.NewTlsConfigFromFlags()
		if common.Error(err) {
			return err
		}

		ecco.httpServer.TLSConfig = tlsConfig
	}

	ecco.ecco.DisableHTTP2 = true
	ecco.ecco.HideBanner = true
	ecco.ecco.HidePort = true
	ecco.ecco.HTTPErrorHandler = func(err error, context echo.Context) {
		common.Error(err)

		echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	ecco.ecco.Use(middleware.LoggerWithConfig(loggerConfig))
	ecco.ecco.Use(middleware.Recover())
	ecco.ecco.Use(middleware.Secure())
	ecco.ecco.Use(middleware.CORS())
	ecco.ecco.Use(session.Middleware(ecco.store))

	return nil
}

func (web *EchoCfg) Close() error {
	common.Info("Web stop")

	return nil
}
