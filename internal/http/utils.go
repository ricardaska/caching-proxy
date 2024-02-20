package httphandler

import (
	"bytes"
	"io"
	"log/slog"
	"maps"
	"net/http"
)

func CopyHeaders(dst http.Header, src http.Header) {
	maps.Copy(dst, src)
}

func CopyResponse(response *http.Response, writer http.ResponseWriter) error {
	CopyHeaders(writer.Header(), response.Header)
	writer.WriteHeader(response.StatusCode)
	if response.Body != nil {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}
		if len(body) > 0 {
			writer.Write(body)
		}
	}
	return nil
}

func ForwardRequest(client *http.Client, path string, request *http.Request, callback func(response *http.Response) error) (err error) {
	var body []byte
	if request.Body != nil {
		body, err = io.ReadAll(request.Body)
		request.Body.Close()
	}

	if err != nil {
		return err
	}

	forwardRequest, err := http.NewRequest(request.Method, path, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	slog.Debug(
		"Forwarding request",
		"method", request.Method,
		"host", forwardRequest.Host,
		"path", forwardRequest.URL.Path,
		"body", string(body),
	)

	CopyHeaders(forwardRequest.Header, request.Header)

	response, err := client.Do(forwardRequest)
	if response != nil && response.Body != nil {
		defer response.Body.Close()
	}

	if err != nil {
		return err
	}

	return callback(response)
}
