package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	stdlog "log"

	nested "github.com/antonfisher/nested-logrus-formatter"
	loglib "github.com/fatedier/golib/log"

	"github.com/fatedier/frp/pkg/util/log"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetFormatter(&nested.Formatter{
		TimestampFormat: "2006-01-02 15:04:05.000",
		FieldsOrder:     []string{"module"},
	})

	log.Logger = loglib.New(
		loglib.WithLevel(log.InfoLevel),
		loglib.WithOutput(loglib.NewConsoleWriter(loglib.ConsoleConfig{Colorful: false}, &frpsLogWriter{})),
	)
	gin.SetMode(gin.ReleaseMode)
	stdlog.SetOutput(&reverseProxyLogger{})
}

func main() {
	frpSvr, err := newFrpsInstance(getAccessToken())
	if err != nil {
		logrus.Panicf("failed to create FRP server instance: %v", err)
	}

	port, _, err := getOutboundPort()
	if err != nil {
		logrus.Panicf("failed to get outbound port: %v", err)
	}
	httpSvr, err := newGinInstance(GIN_OUTBOUND_BIND_ADDR, port)
	if err != nil {
		logrus.Panicf("failed to create gin instance: %v", err)
	}

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		logrus.Infof("starting http server on %s:%s", GIN_OUTBOUND_BIND_ADDR, port)
		if err := httpSvr.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Errorf("failed to start http server: %v", err)
		}
	}()

	go func() {
		frpSvr.Run(context.Background())
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer signal.Stop(signalChan)

	<-signalChan
	logrus.Warnln("interruption signal received, shutting down...")

	httpCtx, httpCancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	if err := httpSvr.Shutdown(httpCtx); err != nil {
		logrus.Errorf("failed to shutdown http server: %v", err)
	}
	httpCancelFn()
	wg.Wait()

	if err := frpSvr.Close(); err != nil {
		logrus.Errorf("failed to close frp server: %v", err)
	}
}
