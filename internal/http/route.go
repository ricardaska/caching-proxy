package httphandler

import (
	"net/http"
	"strings"
)

type Route struct {
	Path          string
	Attachment    any
	HandleRequest http.HandlerFunc
}

func (r *Route) Matches(path string) bool {
	return strings.HasPrefix(path, r.Path)
}
