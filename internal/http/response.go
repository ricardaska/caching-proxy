package httphandler

import (
	"fmt"
	"net/http"
)

type StoredResponse struct {
	Body       []byte
	header     http.Header
	Status     string
	StatusCode int
}

func (response *StoredResponse) Header() http.Header {
	if response.header == nil {
		response.header = make(http.Header)
	}
	return response.header
}

func (response *StoredResponse) Write(body []byte) (int, error) {
	response.Body = body
	return len(body), nil
}

func (response *StoredResponse) WriteHeader(statusCode int) {
	response.StatusCode = statusCode
	response.Status = fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode))
}

func (response *StoredResponse) WriteResponse(writer http.ResponseWriter) {
	CopyHeaders(writer.Header(), response.header)
	writer.WriteHeader(response.StatusCode)
	if response.Body != nil {
		writer.Write(response.Body)
	}
}

func (response *StoredResponse) Clone() *StoredResponse {
	var cloned StoredResponse
	copy(response.Body, cloned.Body)
	CopyHeaders(cloned.header, response.header)
	cloned.Status = response.Status
	cloned.StatusCode = response.StatusCode
	return &cloned
}
