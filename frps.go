package main

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/fatedier/frp/pkg/config/types"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/server"
)

type frpsLogWriter struct{}

func (w *frpsLogWriter) Write(p []byte) (n int, err error) {
	logText := string(p)
	logText = strings.TrimSuffix(logText, "\n")

	if len(logText) < 28 {
		return len(p), nil
	}

	switch logText[25:26] {
	case "I":
		logrus.Infof("FRP server returned info: %s", logText[28:])
	case "W":
		logrus.Warnf("FRP server returned warning: %s", logText[28:])
	case "E":
		logrus.Errorf("FRP server returned error: %s", logText[28:])
	}

	return len(p), nil
}

func newFrpsInstance(token string) (*server.Service, error) {
	trueVal := true

	authConfig := v1.AuthServerConfig{Token: token}
	if err := authConfig.Complete(); err != nil {
		return nil, fmt.Errorf("failed to complete auth config: %w", err)
	}

	transportConfig := v1.ServerTransportConfig{
		TCPMux:                  &trueVal,
		MaxPoolCount:            10,
		TCPMuxKeepaliveInterval: 30,
		TCPKeepAlive:            60,
		HeartbeatTimeout:        90,
	}
	transportConfig.Complete()

	frpsEntryPort := getFrpsEntryPort()
	frpsProxyPort := getFrpsProxyPort()

	commonConfig := &v1.ServerConfig{
		Auth:              authConfig,
		BindAddr:          FRPS_ENTRY_BIND_ADDR,
		BindPort:          frpsEntryPort,
		ProxyBindAddr:     FRPS_PROXY_BIND_ADDR,
		MaxPortsPerClient: 1,
		Transport:         transportConfig,
		AllowPorts:        []types.PortsRange{{Single: frpsProxyPort}},
	}
	if err := commonConfig.Complete(); err != nil {
		return nil, fmt.Errorf("failed to complete common config: %w", err)
	}

	svr, err := server.NewService(commonConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create FRP client instance: %w", err)
	}

	return svr, nil
}
