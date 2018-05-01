package internal

import (
	"context"
	"net/http"
	"runtime"
	"strings"

	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/timer"
	libhoney "github.com/honeycombio/libhoney-go"
	uuid "github.com/satori/go.uuid"
)

type ResponseWriter struct {
	http.ResponseWriter
	Status int
}

func (h *ResponseWriter) WriteHeader(statusCode int) {
	h.Status = statusCode
	h.ResponseWriter.WriteHeader(statusCode)
}

func AddRequestProps(req *http.Request, ev *libhoney.Event) {
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
	ev.AddField("trace.trace_id", traceID)

	// add a span ID
	id, _ := uuid.NewV4()
	ev.AddField("trace.span_id", id.String())
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
		id, _ := uuid.NewV4()
		traceID = id.String()
	}
	return traceID
}

// BuildDBEvent tries to bring together most of the things that need to happen
// for an event to wrap a DB call in bot the sql and sqlx packages. It returns a
// function which, when called, dispatches the event that it created. This lets
// it finish a timer around the call automatically.
func BuildDBEvent(ctx context.Context, bld *libhoney.Builder, query string, args ...interface{}) (*libhoney.Event, func(error)) {
	ev := bld.NewEvent()
	timer := timer.Start()
	fn := func(err error) {
		duration := timer.Finish()
		ev.AddField("duration_ms", duration)
		if err != nil {
			ev.AddField("error", err)
		}
		ev.Send()
	}
	addTraceID(ctx, ev)

	// get the name of the function that called this one. Strip the package and type
	pc, _, _, _ := runtime.Caller(1)
	callName := runtime.FuncForPC(pc).Name()
	callNameChunks := strings.Split(callName, ".")
	ev.AddField("db.call", callNameChunks[len(callNameChunks)-1])

	if query != "" {
		ev.AddField("db.query", query)
	}
	if args != nil {
		ev.AddField("db.query_args", args)
	}
	return ev, fn
}

func addTraceID(ctx context.Context, ev *libhoney.Event) {
	// get a transaction ID from the request's event, if it's sitting in context
	if parentEv := beeline.ContextEvent(ctx); parentEv != nil {
		if id, ok := parentEv.Fields()["trace.trace_id"]; ok {
			ev.AddField("trace.trace_id", id)
		}
	}
}
