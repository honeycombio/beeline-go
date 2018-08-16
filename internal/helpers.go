package internal

import (
	"context"
	"net/http"
	"runtime"
	"strings"

	"github.com/felixge/httpsnoop"
	"github.com/google/uuid"
	"github.com/honeycombio/beeline-go/timer"
	libhoney "github.com/honeycombio/libhoney-go"
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
	reqProps["meta.type"] = "http"
	// Add a variety of details about the HTTP request, such as user agent
	// and method, to any created libhoney event.
	reqProps["request.method"] = req.Method
	reqProps["request.path"] = req.URL.Path
	reqProps["request.host"] = req.URL.Host
	reqProps["request.http_version"] = req.Proto
	reqProps["request.content_length"] = req.ContentLength
	reqProps["request.remote_addr"] = req.RemoteAddr
	reqProps["request.header.user_agent"] = req.UserAgent()
	return reqProps
}

// FindTraceHeaders parses tracing headers if they exist. Uses beeline headers
// first, then looks for others.
//
// Request-Id: abcd-1234-uuid-v4
// X-Amzn-Trace-Id X-Amzn-Trace-Id: Self=1-67891234-12456789abcdef012345678;Root=1-67891233-abcdef012345678912345678;CalledFrom=app
//
// adds all trace IDs to the passed in event, and returns a trace ID if it finds
// one. Request-ID is preferred over the Amazon trace ID. Will generate a UUID
// if it doesn't find any trace IDs.
//
// NOTE that Amazon actually only means for the latter part of the header to be
// the ID - format is version-timestamp-id. For now though (TODO) we treat it as
// the entire string
//
// If getting trace context from another beeline, also returns any fields
// included to be added to the trace as Trace level fields
func FindTraceHeaders(req *http.Request) (*TraceHeader, map[string]interface{}, error) {
	beelineHeader := req.Header.Get(TracePropagationHTTPHeader)
	if beelineHeader != "" {
		return UnmarshalTraceContext(beelineHeader)
	}
	// didn't find it from a beeline, let's go looking elsewhere
	headers := &TraceHeader{}
	var traceID string
	awsHeader := req.Header.Get("X-Amzn-Trace-Id")
	if awsHeader != "" {
		headers.Source = HeaderSourceAmazon
		// break into key=val pairs on `;` and add each key=val header
		ids := strings.Split(awsHeader, ";")
		for _, id := range ids {
			keyval := strings.Split(id, "=")
			if len(keyval) != 2 {
				// malformed keyval
				continue
			}
			// ev.AddField("request.header.aws_trace_id."+keyval[0], keyval[1])
			if keyval[0] == "Root" {
				traceID = keyval[1]
			}
		}
	}
	requestID := req.Header.Get("Request-Id")
	if requestID != "" {
		headers.Source = HeaderSourceBeeline
		// ev.AddField("request.header.request_id", requestID)
		traceID = requestID
	}
	if traceID == "" {
		id, _ := uuid.NewRandom()
		traceID = id.String()
	}
	headers.TraceID = traceID
	return headers, nil, nil
}

// BuildDBEvent tries to bring together most of the things that need to happen
// for an event to wrap a DB call in bot the sql and sqlx packages. It returns a
// function which, when called, dispatches the event that it created. This lets
// it finish a timer around the call automatically. This function is only used
// when no context is available to the caller - if context is available, use
// BuildDBSpan() instead.
func BuildDBEvent(bld *libhoney.Builder, query string, args ...interface{}) (*libhoney.Event, func(error)) {
	timer := timer.Start()
	ev := bld.NewEvent()
	fn := func(err error) {
		duration := timer.Finish()
		// rollup(ctx, ev, duration)
		ev.AddField("duration_ms", duration)
		if err != nil {
			ev.AddField("db.error", err)
		}
		ev.Metadata, _ = ev.Fields()["name"]
		ev.Send()
	}

	// get the name of the function that called this one. Strip the package and type
	pc, _, _, _ := runtime.Caller(1)
	callName := runtime.FuncForPC(pc).Name()
	callNameChunks := strings.Split(callName, ".")
	ev.AddField("db.call", callNameChunks[len(callNameChunks)-1])
	ev.AddField("name", callNameChunks[len(callNameChunks)-1])

	if query != "" {
		ev.AddField("db.query", query)
	}
	if args != nil {
		ev.AddField("db.query_args", args)
	}
	return ev, fn
}

// BuildDBSpan does the same things as BuildDBEvent except that it has access to
// a trace from the context and takes advantage of that to add the DB events
// into the trace.
func BuildDBSpan(ctx context.Context, bld *libhoney.Builder, query string, args ...interface{}) func(error) {
	timer := timer.Start()
	ev := bld.NewEvent()
	trace := GetTraceFromContext(ctx)
	if trace == nil || trace.shouldDrop {
		// if we have no trace or we're supposed to drop this trace, return a noop fn
		return func(err error) {}
	}
	var span *Span
	ctx, span = StartSpanWithEvent(ctx, ev)
	fn := func(err error) {
		duration := timer.Finish()
		if err != nil {
			ev.AddField("db.error", err)
		}
		span.AddRollupField("db.duration_ms", duration)
		span.AddRollupField("db.call_count", 1)
		FinishSpan(ctx)
	}
	// get the name of the function that called this one. Strip the package and type
	pc, _, _, _ := runtime.Caller(1)
	callName := runtime.FuncForPC(pc).Name()
	callNameChunks := strings.Split(callName, ".")
	ev.AddField("db.call", callNameChunks[len(callNameChunks)-1])
	ev.AddField("name", callNameChunks[len(callNameChunks)-1])

	if query != "" {
		ev.AddField("db.query", query)
	}
	if args != nil {
		ev.AddField("db.query_args", args)
	}
	return fn
}
