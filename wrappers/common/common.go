package common

import (
	"context"
	"database/sql"
	"fmt"
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
	// Wrapped is not embedded to prevent ResponseWriter from directly
	// fulfilling the http.ResponseWriter interface. Wrapping in this
	// way would obscure optional http.ResponseWriter interfaces.
	Wrapped http.ResponseWriter
	Status  int
}

func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	var rw ResponseWriter

	rw.Wrapped = httpsnoop.Wrap(w, httpsnoop.Hooks{
		WriteHeader: func(next httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
			return func(code int) {
				// The first call to WriteHeader sends the response header.
				// Any subsequent calls are invalid. Only record the first
				// code written.
				if rw.Status == 0 {
					rw.Status = code
				}
				next(code)
			}
		},
	})

	return &rw
}

// StartSpanOrTraceFromHTTP looks at the incoming HTTP request to see if there's
// a trace header and starts a new trace using the propagation header if it
// exists.
func StartSpanOrTraceFromHTTP(r *http.Request) (context.Context, *trace.Span) {
	return StartSpanOrTraceFromHTTPDelegateHeader(r, nil)
}

// StartSpanOrTraceFromHTTPDelegateHeader passes the incoming HTTP request to a
// callback to see if there's a trace header and starts a new trace using the
// propagation header if it exists. If the fetch header callback is nil it will
// use the default beeline headers.
func StartSpanOrTraceFromHTTPDelegateHeader(r *http.Request, fetchTraceHeader func(*http.Request) (*propagation.Propagation, error)) (context.Context, *trace.Span) {
	ctx := r.Context()
	span := trace.GetSpanFromContext(ctx)
	if span == nil {
		// there is no trace yet. We should make one! and use the root span.
		if fetchTraceHeader == nil {
			fetchTraceHeader = GetPropFromBeelineHeader
		}
		prop, _ := fetchTraceHeader(r)
		var tr *trace.Trace
		ctx, tr = trace.NewTraceFromPropagation(ctx, prop)
		span = tr.GetRootSpan()
	} else {
		// we had a parent! let's make a new child for this handler
		ctx, span = span.CreateChild(ctx)
	}
	// go get any common HTTP headers and attributes to add to the span
	for k, v := range GetRequestProps(r) {
		span.AddField(k, v)
	}
	return ctx, span
}

// IgnoreTraceHeaders will fulfill the fetchHeaders syntax but never return trace
// headers
func IgnoreTraceHeaders(r *http.Request) (*propagation.Propagation, error) {
	return nil, nil
}

// GetPropFromBeelineHeader will return a trace propogataion object based on the
// default beeline headers
func GetPropFromBeelineHeader(r *http.Request) (*propagation.Propagation, error) {
	beelineHeader := r.Header.Get(propagation.TracePropagationHTTPHeader)
	if beelineHeader == "" {
		return nil, fmt.Errorf("beeline header absent")
	}
	return propagation.UnmarshalTraceContext(beelineHeader)
}

// GetPropFromAWSHeader will return a trace propogataion object based on the
// headers created by AWS ELB and ALBs
func GetPropFromAWSHeader(r *http.Request) (*propagation.Propagation, error) {
	// docs on how to parse an AWS trace header
	// https://docs.aws.amazon.com/elasticloadbalancing/latest/application/load-balancer-request-tracing.html
	awsHeader := r.Header.Get("X-Amzn-Trace-Id")
	if awsHeader == "" {
		return nil, fmt.Errorf("aws trace header absent")
	}
	return propagation.UnmarshalAWSTraceContext(awsHeader)
}

// GetRequestProps is a convenient method to grab all common http request
// properties and get them back as a map.
func GetRequestProps(req *http.Request) map[string]interface{} {
	userAgent := req.UserAgent()
	xForwardedFor := req.Header.Get("x-forwarded-for")
	xForwardedProto := req.Header.Get("x-forwarded-proto")

	reqProps := make(map[string]interface{})
	// identify the type of event
	reqProps["meta.type"] = "http_request"
	// Add a variety of details about the HTTP request, such as user agent
	// and method, to any created libhoney event.
	reqProps["request.method"] = req.Method
	reqProps["request.path"] = req.URL.Path
	if req.URL.RawQuery != "" {
		reqProps["request.query"] = req.URL.RawQuery
	}
	reqProps["request.url"] = req.URL.String()
	reqProps["request.host"] = req.Host
	reqProps["request.http_version"] = req.Proto
	reqProps["request.content_length"] = req.ContentLength
	reqProps["request.remote_addr"] = req.RemoteAddr
	if userAgent != "" {
		reqProps["request.header.user_agent"] = userAgent
	}
	if xForwardedFor != "" {
		reqProps["request.header.x_forwarded_for"] = xForwardedFor
	}
	if xForwardedProto != "" {
		reqProps["request.header.x_forwarded_proto"] = xForwardedProto
	}
	return reqProps
}

// getCallersNames grabs the current call stack, skips up a few levels, then
// grabs as many function names as depth. Suggested use is something like 1, 2
// meaning "get my parent and its parent". skip=0 means the function calling
// this one.
func getCallersNames(skip, depth int) []string {
	callers := make([]string, 0, depth)
	callerPcs := make([]uintptr, depth)
	// add 2 to skip to account for runtime.Callers and getCallersNames
	numCallers := runtime.Callers(skip+2, callerPcs)
	// If there are no callers, the entire stacktrace is nil
	if numCallers == 0 {
		return callers
	}
	callersFrames := runtime.CallersFrames(callerPcs)
	for i := 0; i < depth; i++ {
		fr, more := callersFrames.Next()
		// store the function's name
		nameParts := strings.Split(fr.Function, ".")
		callers = append(callers, nameParts[len(nameParts)-1])
		if !more {
			break
		}
	}
	return callers
}

func sharedDBEvent(bld *libhoney.Builder, query string, args ...interface{}) *libhoney.Event {
	ev := bld.NewEvent()

	// skip 2 - this one and the buildDB*, so we get the sqlx function and its parent
	callerNames := getCallersNames(2, 2)
	switch len(callerNames) {
	case 2:
		ev.AddField("db.call", callerNames[0])
		ev.AddField("name", callerNames[0])
		ev.AddField("db.caller", callerNames[1])
	case 1:
		ev.AddField("db.call", callerNames[0])
		ev.AddField("name", callerNames[0])
	default:
		ev.AddField("name", "db")
	}

	if query != "" {
		ev.AddField("db.query", query)
	}
	if args != nil {
		ev.AddField("db.query_args", args)
	}
	return ev
}

// BuildDBEvent tries to bring together most of the things that need to happen
// for an event to wrap a DB call in both the sql and sqlx packages. It returns a
// function which, when called, dispatches the event that it created. This lets
// it finish a timer around the call automatically. This function is only used
// when no context (and therefore no beeline trace) is available to the caller -
// if context is available, use BuildDBSpan() instead to tie it in to the active
// trace.
func BuildDBEvent(bld *libhoney.Builder, stats sql.DBStats, query string, args ...interface{}) (*libhoney.Event, func(error)) {
	timer := timer.Start()
	ev := sharedDBEvent(bld, query, args)
	addDBStatsToEvent(ev, stats)
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
	return ev, fn
}

// BuildDBSpan does the same things as BuildDBEvent except that it has access to
// a trace from the context and takes advantage of that to add the DB events
// into the trace.
func BuildDBSpan(ctx context.Context, bld *libhoney.Builder, stats sql.DBStats, query string, args ...interface{}) (context.Context, *trace.Span, func(error)) {
	timer := timer.Start()
	parentSpan := trace.GetSpanFromContext(ctx)
	var span *trace.Span
	if parentSpan == nil {
		// if we have no trace, make a new one. This is unfortunate but the
		// least confusing possibility. Would be nice to indicate this had
		// happened in a better way than yet another meta. field.
		var tr *trace.Trace
		ctx, tr = trace.NewTrace(ctx, "")
		span = tr.GetRootSpan()
		span.AddField("meta.orphaned", true)
	} else {
		ctx, span = parentSpan.CreateChild(ctx)
	}
	addDBStatsToSpan(span, stats)

	ev := sharedDBEvent(bld, query, args...)
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
		span.Send()
	}
	return ctx, span, fn
}
