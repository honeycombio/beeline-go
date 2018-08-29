package common

import (
	"context"
	"net/http"
	"runtime"
	"strings"

	"github.com/felixge/httpsnoop"
	"github.com/honeycombio/beeline-go/propagation"
	"github.com/honeycombio/beeline-go/timer"
	"github.com/honeycombio/beeline-go/trace"
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

func StartSpanOrTraceFromHTTP(r *http.Request) (context.Context, *trace.Span) {
	ctx := r.Context()
	span := trace.GetSpanFromContext(ctx)
	if span == nil {
		// there is no trace yet. We should make one! and use the root span.
		beelineHeader := r.Header.Get(propagation.TracePropagationHTTPHeader)
		var tr *trace.Trace
		ctx, tr = trace.NewTrace(ctx, beelineHeader)
		span = tr.GetRootSpan()
	} else {
		// we had a parent! let's make a new child for this handler
		ctx, span = span.ChildSpan(ctx)
	}
	// go get any common HTTP headers and attributes to add to the span
	for k, v := range GetRequestProps(r) {
		span.AddField(k, v)
	}
	return ctx, span
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
	reqProps["request.header.x_forwarded_for"] = req.Header.Get("x-forwarded-for")
	reqProps["request.header.x_forwarded_proto"] = req.Header.Get("x-forwarded-proto")
	return reqProps
}

// BuildDBEvent tries to bring together most of the things that need to happen
// for an event to wrap a DB call in bot the sql and sqlx packages. It returns a
// function which, when called, dispatches the event that it created. This lets
// it finish a timer around the call automatically. This function is only used
// when no context (and therefore no beeline trace) is available to the caller -
// if context is available, use BuildDBSpan() instead to tie it in to the active
// trace.
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
func BuildDBSpan(ctx context.Context, bld *libhoney.Builder, query string, args ...interface{}) (context.Context, *trace.Span, func(error)) {
	timer := timer.Start()
	parentSpan := trace.GetSpanFromContext(ctx)
	if parentSpan == nil {
		// if we have no trace or we're supposed to drop this trace, return a noop fn
		return ctx, nil, func(err error) {}
	}
	ctx, span := parentSpan.ChildSpan(ctx)

	ev := bld.NewEvent()
	for k, v := range ev.Fields() {
		span.AddField(k, v)
	}
	fn := func(err error) {
		duration := timer.Finish()
		if err != nil {
			span.AddField("db.error", err)
		}
		span.AddRollupField("db.duration_ms", duration)
		span.AddRollupField("db.call_count", 1)
		span.Finish()
	}
	// get the name of the function that called this one. Strip the package and type
	pc, _, _, _ := runtime.Caller(1)
	callName := runtime.FuncForPC(pc).Name()
	callNameChunks := strings.Split(callName, ".")
	span.AddField("db.call", callNameChunks[len(callNameChunks)-1])
	span.AddField("name", callNameChunks[len(callNameChunks)-1])

	if query != "" {
		span.AddField("db.query", query)
	}
	if args != nil {
		span.AddField("db.query_args", args)
	}
	return ctx, span, fn
}
