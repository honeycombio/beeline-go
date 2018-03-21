package honeycomb

import (
	"context"
	"net/http"
	"strings"

	libhoney "github.com/honeycombio/libhoney-go"
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
	// Add a variety of details about the HTTP request, such as user agent
	// and method, to any created libhoney event.
	ev.AddField("request.method", req.Method)
	ev.AddField("request.path", req.URL.Path)
	ev.AddField("request.host", req.URL.Host)
	ev.AddField("request.proto", req.Proto)
	ev.AddField("request.content_length", req.ContentLength)
	ev.AddField("request.remote_addr", req.RemoteAddr)
	ev.AddField("request.user_agent", req.UserAgent())
	// add any AWS trace headers that might be present
	parseAWSTraceHeader(req, ev)
	// TODO add other trace headers

}

// parse tracing headers if they exist X-Amzn-Trace-Id
// X-Amzn-Trace-Id: Self=1-67891234-12456789abcdef012345678;Root=1-67891233-abcdef012345678912345678;CalledFrom=app
func parseAWSTraceHeader(req *http.Request, ev *libhoney.Event) {
	traceHeader := req.Header.Get("X-Amzn-Trace-Id")
	if traceHeader == "" {
		// no header found
		return
	}
	// break into key=val pairs on `;` and add each key=val header
	ids := strings.Split(traceHeader, ";")
	for _, id := range ids {
		keyval := strings.Split(id, "=")
		if len(keyval) != 2 {
			// malformed keyval
			continue
		}
		ev.AddField("request.trace_id."+keyval[0], keyval[1])
	}
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
