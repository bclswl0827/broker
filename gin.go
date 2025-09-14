package main

import (
	"fmt"
	"math"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/koding/websocketproxy"
	"github.com/sirupsen/logrus"
)

type reverseProxyLogger struct{}

func (w *reverseProxyLogger) Write(p []byte) (n int, err error) {
	logText := string(p)
	logText = strings.TrimSuffix(logText, "\n")

	if len(logText) < 20 {
		return len(p), nil
	}

	logrus.Infof("reverse proxy returned: %s", logText[20:])
	return len(p), nil
}

type reverseProxyOptions struct {
	Target      string
	PathRewrite string
}

func newGinReverseProxy(path string, option reverseProxyOptions) gin.HandlerFunc {
	targetUrl, _ := url.Parse(option.Target)
	httpProxy := httputil.NewSingleHostReverseProxy(targetUrl)

	wsTargetUrl := *targetUrl
	wsTargetUrl.Scheme = "ws"
	wsProxy := websocketproxy.NewProxy(&wsTargetUrl)
	wsProxy.Upgrader = &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	return func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.RequestURI, path) {
			c.Next()
			return
		}

		c.Request.URL.Path = strings.Replace(c.Request.URL.Path, option.PathRewrite, "", 1)
		c.Request.Host = targetUrl.Host

		if strings.EqualFold(c.GetHeader("Upgrade"), "websocket") {
			wsProxy.ServeHTTP(c.Writer, c.Request)
		} else {
			httpProxy.ServeHTTP(c.Writer, c.Request)
		}

		c.Abort()
	}
}

func newGinHttpLogger(notLogged ...string) gin.HandlerFunc {
	var skip map[string]struct{}

	if length := len(notLogged); length > 0 {
		skip = make(map[string]struct{}, length)

		for _, p := range notLogged {
			skip[p] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		start := time.Now()
		c.Next()
		stop := time.Since(start)
		latency := int(math.Ceil(float64(stop.Nanoseconds()) / 1000000.0))
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		clientUserAgent := c.Request.UserAgent()

		if _, ok := skip[path]; ok {
			return
		}

		if len(c.Errors) > 0 {
			logrus.Error(c.Errors.ByType(gin.ErrorTypePrivate).String())
		} else {
			msg := fmt.Sprintf("%s - \"%s %s\" %d \"%s\" (%d ms)", clientIP, c.Request.Method, path, statusCode, clientUserAgent, latency)
			if statusCode >= http.StatusInternalServerError {
				logrus.Error(msg)
			} else if statusCode >= http.StatusBadRequest {
				logrus.Warn(msg)
			} else {
				logrus.Info(msg)
			}
		}
	}
}

func newGinInstance(addr, port string) (*http.Server, error) {
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(newGinHttpLogger())
	engine.Use(newGinReverseProxy("/~!frp", reverseProxyOptions{
		Target: fmt.Sprintf("http://%s:%d/~!frp", FRPS_ENTRY_BIND_ADDR, FRPS_ENTRY_BIND_PORT),
	}))
	engine.Use(newGinReverseProxy("/", reverseProxyOptions{
		Target: fmt.Sprintf("http://%s:%d", FRPS_PROXY_BIND_ADDR, FRPS_PROXY_BIND_PORT),
	}))

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", addr, port),
		Handler: engine,
	}
	return server, nil
}
