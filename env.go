package main

import (
	"os"
	"strconv"
)

func getFrpsEntryPort() int {
	portStr := os.Getenv("FRPS")
	if len(portStr) == 0 {
		return FRPS_ENTRY_DEFAULT_PORT
	}

	portNum, err := strconv.Atoi(portStr)
	if err != nil {
		return FRPS_ENTRY_DEFAULT_PORT
	}

	return portNum
}

func getFrpsProxyPort() int {
	portStr := os.Getenv("PROXY")
	if len(portStr) == 0 {
		return FRPS_PROXY_DEFAULT_PORT
	}

	portNum, err := strconv.Atoi(portStr)
	if err != nil {
		return FRPS_PROXY_DEFAULT_PORT
	}

	return portNum
}

func getOutboundPort() (string, int, error) {
	portStr := os.Getenv("PORT")
	if len(portStr) == 0 {
		portStr = "8080"
	}

	portNum, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, err
	}

	return portStr, portNum, nil
}

func getAccessToken() string {
	token := os.Getenv("TOKEN")
	if len(token) == 0 {
		token = "hello-frps"
	}

	return token
}
