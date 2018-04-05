package honeycomb

import (
	"context"
	"net/http"
	"strings"

	libhoney "github.com/honeycombio/libhoney-go"
	uuid "github.com/satori/go.uuid"
)

const honeyBuilderContextKey = "honeycombBuilderContextKey"
const honeyEventContextKey = "honeycombEventContextKey"

type hnyResponseWriter struct {
	http.ResponseWriter
	status int
}

func (h *hnyResponseWriter) WriteHeader(statusCode int) {
	h.status = statusCode
	h.ResponseWriter.WriteHeader(statusCode)
}

func addRequestProps(req *http.Request, ev *libhoney.Event) {
	// identify the type of event
	ev.AddField("meta.type", "http request")
	// Add a variety of details about the HTTP request, such as user agent
	// and method, to any created libhoney event.
	ev.AddField("request.method", req.Method)
	ev.AddField("request.path", req.URL.Path)
	ev.AddField("request.host", req.URL.Host)
	ev.AddField("request.proto", req.Proto)
	ev.AddField("request.content_length", req.ContentLength)
	ev.AddField("request.remote_addr", req.RemoteAddr)
	ev.AddField("request.header.user_agent", req.UserAgent())
	// add any AWS trace headers that might be present
	traceID := parseTraceHeader(req, ev)
	ev.AddField("Trace.TraceId", traceID)
}

// parseTraceHeader parses tracing headers if they exist
//
// Request-Id: abcd-1234-uuid-v4
// X-Amzn-Trace-Id X-Amzn-Trace-Id: Self=1-67891234-12456789abcdef012345678;Root=1-67891233-abcdef012345678912345678;CalledFrom=app
//
// adds all trace IDs to the passed in event, and returns a trace ID if it finds
// one. Request-ID is preferred over the Amazon trace ID. Will generate a UUID
// if it doesn't find any trace IDs.
func parseTraceHeader(req *http.Request, ev *libhoney.Event) string {
	var traceID string
	awsHeader := req.Header.Get("X-Amzn-Trace-Id")
	if awsHeader != "" {
		// break into key=val pairs on `;` and add each key=val header
		ids := strings.Split(awsHeader, ";")
		for _, id := range ids {
			keyval := strings.Split(id, "=")
			if len(keyval) != 2 {
				// malformed keyval
				continue
			}
			ev.AddField("request.header.aws_trace_id."+keyval[0], keyval[1])
			if keyval[0] == "Root" {
				traceID = keyval[0]
			}
		}
	}
	requestID := req.Header.Get("Request-Id")
	if requestID != "" {
		ev.AddField("request.header.request_id", requestID)
		traceID = requestID
	}
	if traceID == "" {
		traceID = uuid.NewV4().String()
	}
	return traceID
}

// might return a thing or nil if there wasn't one there already
func existingEventFromContext(ctx context.Context) *libhoney.Event {
	if evt, ok := ctx.Value(honeyEventContextKey).(*libhoney.Event); ok {
		return evt
	}
	return nil
}

// might return a thing or nil if there wasn't one there already
func existingBuilderFromContext(ctx context.Context) *libhoney.Builder {
	if bldr, ok := ctx.Value(honeyBuilderContextKey).(*libhoney.Builder); ok {
		return bldr
	}
	return nil
}
