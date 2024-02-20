package main

import (
	"flag"
	"log/slog"
	"net"
	"net/http"
	"os"
	"proxy/internal/caching"
	"proxy/internal/ctrl"
	ihttp "proxy/internal/http"
)

var configPath = flag.String("config", "config.toml", "configuration file path")

var logLevel slog.LevelVar

func main() {
	initLogger()

	flag.Parse()

	config, err := ReadConfig(*configPath)
	if err != nil {
		slog.Error("Failed to read config file", "error", err)
		os.Exit(1)
	}

	logLevel.Set(config.LogLevel)

	httpClient := InitHTTPClient(config.HTTPClient)

	var controlServer *ctrl.Server
	if config.ControlServer.Enabled {
		controlServer = ctrl.NewServer(config.ControlServer.Network, config.ControlServer.BindAddr)
	}

	proxy := NewProxy(httpClient, controlServer)

	for _, serverConfig := range config.Servers {
		server := proxy.GetServer(serverConfig.BindAddr)

		slog.Info(
			"Configuring HTTP server",
			"address", serverConfig.BindAddr,
		)

		for _, route := range serverConfig.Routes {
			timeToLive := route.TTL.Duration()

			slog.Info(
				"Adding route",
				"target", route.Target,
				"path", route.Path,
				"ttl", timeToLive,
			)

			var cache *caching.Cache[*ihttp.StoredResponse]
			if timeToLive > 0 {
				cache = caching.NewCache[*ihttp.StoredResponse](timeToLive)
			}

			responseHandler := HandlerChain(
				AlterHeaders(route.KeepHeaders, route.DropHeaders),
				ihttp.CopyResponse,
			)

			server.Handle(route.Path, cache, func(writer http.ResponseWriter, request *http.Request) {
				proxy.Handle(route.Path, route.Target, cache, responseHandler, writer, request)
			})
		}
	}

	proxy.Run()
}

func initLogger() {
	logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: &logLevel,
	})
	slog.SetDefault(slog.New(logHandler))
}

func InitHTTPClient(config *HTTPClientConfig) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSHandshakeTimeout = config.TLSTimeout.Duration()
	transport.ResponseHeaderTimeout = config.HeaderTimeout.Duration()
	transport.IdleConnTimeout = config.IdleTimeout.Duration()
	transport.MaxIdleConnsPerHost = config.MaxIdleConns
	transport.DialContext = (&net.Dialer{
		Timeout: config.TCPTimeout.Duration(),
	}).DialContext

	client := &http.Client{
		Transport: transport,
	}

	return client
}
