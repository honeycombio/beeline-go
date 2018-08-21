package common

import (
	"net/http"

	"github.com/felixge/httpsnoop"
)

type ResponseWriter struct {
	http.ResponseWriter
	Status int
}

func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: httpsnoop.Wrap(w, httpsnoop.Hooks{}),
	}
}

func (h *ResponseWriter) WriteHeader(statusCode int) {
	h.Status = statusCode
	h.ResponseWriter.WriteHeader(statusCode)
}

// GetRequestProps is a convenient method to grab all common http request
// properties and get them back as a map.
func GetRequestProps(req *http.Request) map[string]interface{} {
	reqProps := make(map[string]interface{})
	// identify the type of event
	reqProps["meta.type"] = "http_request"
	// Add a variety of details about the HTTP request, such as user agent
	// and method, to any created libhoney event.
	reqProps["request.method"] = req.Method
	reqProps["request.path"] = req.URL.Path
	reqProps["request.host"] = req.Host
	reqProps["request.http_version"] = req.Proto
	reqProps["request.content_length"] = req.ContentLength
	reqProps["request.remote_addr"] = req.RemoteAddr
	reqProps["request.header.user_agent"] = req.UserAgent()
	return reqProps
}
