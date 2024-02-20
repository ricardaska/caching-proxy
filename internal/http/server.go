package httphandler

import (
	"net/http"
	"sort"
)

type Server struct {
	Address string
	routes  []*Route
}

func NewServer(addr string) *Server {
	return &Server{Address: addr}
}

func (server *Server) AddRoute(route *Route) {
	server.routes = append(server.routes, route)
	sort.Slice(server.routes, func(i, j int) bool {
		return len(server.routes[j].Path) < len(server.routes[i].Path)
	})
}

func (server *Server) Handle(path string, attachment any, handler http.HandlerFunc) {
	route := &Route{Path: path, Attachment: attachment, HandleRequest: handler}
	server.AddRoute(route)
}

func (server *Server) GetRoute(path string) *Route {
	for _, route := range server.routes {
		if !route.Matches(path) {
			continue
		}
		return route
	}
	return nil
}

func (server *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if route := server.GetRoute(request.URL.Path); route != nil {
		route.HandleRequest(writer, request)
	} else {
		writer.WriteHeader(http.StatusNotFound)
	}
}

func (server *Server) ListenAndServe() error {
	return http.ListenAndServe(server.Address, server)
}
