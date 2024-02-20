package main

import (
	"log/slog"
	"net/http"
	"os"
	"proxy/internal/caching"
	"proxy/internal/ctrl"
	ihttp "proxy/internal/http"
	"strings"
	"sync"
)

type ResponseHandler func(response *http.Response, writer http.ResponseWriter) error

type Proxy struct {
	Client        *http.Client
	ControlServer *ctrl.Server
	Servers       map[string]*ihttp.Server
}

func NewProxy(httpClient *http.Client, controlServer *ctrl.Server) *Proxy {
	return &Proxy{Client: httpClient, ControlServer: controlServer, Servers: make(map[string]*ihttp.Server)}
}

func (proxy *Proxy) GetServer(name string) *ihttp.Server {
	if server, ok := proxy.Servers[name]; ok {
		return server
	}
	server := ihttp.NewServer(name)
	proxy.Servers[name] = server
	return server
}

func (proxy *Proxy) Run() {
	if proxy.ControlServer != nil {
		go proxy.runControlServer()
	}

	var wg sync.WaitGroup
	wg.Add(len(proxy.Servers))
	for _, value := range proxy.Servers {
		server := value
		go func() {
			slog.Info("Starting HTTP listener", "address", server.Address)
			if err := server.ListenAndServe(); err != nil {
				slog.Error("HTTP server error", "error", err, "address", server.Address)
				os.Exit(1)
			}
			wg.Done()
		}()
	}

	wg.Wait()
}

func (proxy *Proxy) ForwardRequest(host string, writer http.ResponseWriter, request *http.Request, responseHandler ResponseHandler) error {
	return ihttp.ForwardRequest(proxy.Client, host+request.URL.Path, request, func(response *http.Response) error {
		return responseHandler(response, writer)
	})
}

func (proxy *Proxy) Handle(path, host string, cache *caching.Cache[*ihttp.StoredResponse], responseHandler ResponseHandler, writer http.ResponseWriter, request *http.Request) {
	if cache == nil {
		if err := proxy.ForwardRequest(host, writer, request, responseHandler); err != nil {
			slog.Error("Error occured while forwarding request", "error", err)
		}
		return
	}

	key := GetCacheKey(path, request.URL.Path)

	response := cache.Load(key, func() *ihttp.StoredResponse {
		var storedResponse ihttp.StoredResponse
		if err := proxy.ForwardRequest(host, &storedResponse, request, responseHandler); err != nil {
			storedResponse.WriteHeader(http.StatusInternalServerError)
			slog.Error("Error occured while forwarding request", "error", err)
		}

		slog.Debug(
			"Stored response",
			"host", host,
			"method", request.Method,
			"path", request.URL.Path,
			"status", storedResponse.Status,
			"body", string(storedResponse.Body),
		)

		return &storedResponse
	})

	slog.Debug(
		"Writing response",
		"address", request.RemoteAddr,
		"method", request.Method,
		"status", response.Status,
		"host", request.Host,
		"path", request.URL.Path,
	)

	response.WriteResponse(writer)
}

func (proxy *Proxy) runControlServer() {
	controlServer := proxy.ControlServer

	if controlServer == nil {
		return
	}

	// drop <url path>
	controlServer.AddCommand("drop", func(args []string) error {
		if len(args) == 0 {
			return ctrl.ErrInvalidArguments
		}

		for name, server := range proxy.Servers {
			route := server.GetRoute(args[0])
			if route == nil {
				continue
			}

			cache := route.Attachment.(*caching.Cache[*ihttp.StoredResponse])
			if cache == nil {
				continue
			}

			if removed := cache.Remove(GetCacheKey(route.Path, args[0])); removed != nil {
				slog.Info(
					"Cached response removed",
					"server", name,
					"path", args[0],
				)
			}
		}
		return nil
	})

	// drop_prefix <url path>
	controlServer.AddCommand("drop_prefix", func(args []string) error {
		if len(args) == 0 {
			return ctrl.ErrInvalidArguments
		}

		for name, server := range proxy.Servers {
			route := server.GetRoute(args[0])
			if route == nil {
				continue
			}

			cache := route.Attachment.(*caching.Cache[*ihttp.StoredResponse])
			if cache == nil {
				continue
			}

			prefix := strings.TrimPrefix(args[0], route.Path)

			var removed bool
			cache.RemoveKeyFunc(func(key string) bool {
				if strings.HasPrefix(key, prefix) {
					removed = true
					return true
				}
				return false
			})

			if removed {
				slog.Info(
					"Cached responses removed",
					"server", name,
					"prefix", args[0],
				)
			}
		}
		return nil
	})

	// log_level <new_level>
	controlServer.AddCommand("log_level", func(args []string) error {
		if len(args) == 0 {
			return ctrl.ErrInvalidArguments
		}

		var level slog.Level
		if err := level.UnmarshalText([]byte(args[0])); err != nil {
			return err
		}

		if logLevel.Level() != level {
			logLevel.Set(level)
			slog.Info(
				"Log level changed",
				"level", level,
			)
		}
		return nil
	})

	slog.Info("Starting control server listener", "network", controlServer.Network, "address", controlServer.Address)
	if err := controlServer.Listen(); err != nil {
		slog.Error("Error occured on control server listen", "error", err)
		os.Exit(1)
	}

}

func HandlerChain(handlers ...ResponseHandler) ResponseHandler {
	return func(response *http.Response, writer http.ResponseWriter) error {
		for _, h := range handlers {
			if err := h(response, writer); err != nil {
				return err
			}
		}
		return nil
	}
}

func AlterHeaders(keepHeaders []string, dropHeaders []string) ResponseHandler {
	return func(response *http.Response, writer http.ResponseWriter) error {
		if keepHeaders != nil {
			for k := range response.Header {
				if !IgnoreCaseContains(keepHeaders, k) {
					delete(response.Header, k)
				}
			}
		}
		if dropHeaders != nil {
			for k := range response.Header {
				if IgnoreCaseContains(dropHeaders, k) {
					delete(response.Header, k)
				}
			}
		}
		return nil
	}
}

func GetCacheKey(routePath, requestPath string) string {
	key := strings.TrimPrefix(requestPath, routePath)
	if len(key) == 0 {
		key = requestPath
	}
	return key
}

func IgnoreCaseContains(slice []string, value string) bool {
	for _, v := range slice {
		if strings.EqualFold(v, value) {
			return true
		}
	}
	return false
}
