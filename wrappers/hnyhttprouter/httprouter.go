package hnyhttprouter

import (
	"net/http"
	"reflect"
	"runtime"

	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/internal"
	"github.com/julienschmidt/httprouter"
)

// Middleware wraps httprouter handlers. Since it wraps handlers with explicit
// parameters, it can add those values to the event it generates.
func Middleware(handle httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		var span *internal.Span
		if !beeline.HasTrace(r.Context()) {
			// pick up any trace context from our caller, if present
			traceHeaders, traceContext, _ := internal.FindTraceHeaders(r)
			// use the trace IDs found to spin up a new trace
			ctx, span = internal.StartTraceWithIDs(r.Context(),
				traceHeaders.TraceID, traceHeaders.ParentID, "")
			trace := internal.GetTraceFromContext(ctx)
			// add any additional context to the trace
			for k, v := range traceContext {
				trace.AddField(k, v)
			}
			// and make sure it gets completely sent when we're done.
			defer trace.Send()
		} else {
			// if we're not the root span, just add another layer to our trace.
			ctx, span = internal.StartSpan(r.Context(), "")
		}
		defer span.Finish(ctx)
		// push the context with our trace on to the request
		r = r.WithContext(ctx)
		// go get any common HTTP headers and attributes to add to the span
		for k, v := range internal.GetRequestProps(r) {
			span.AddField(k, v)
		}
		// replace the writer with our wrapper to catch the status code
		wrappedWriter := internal.NewResponseWriter(w)

		// pull out any variables in the URL, add the thing we're matching, etc.
		for _, param := range ps {
			span.AddField("handler.vars."+param.Key, param.Value)
		}
		name := runtime.FuncForPC(reflect.ValueOf(handle).Pointer()).Name()
		span.AddField("handler.name", name)
		span.AddField("name", name)

		handle(w, r, ps)

		if wrappedWriter.Status == 0 {
			wrappedWriter.Status = 200
		}
		span.AddField("response.status_code", wrappedWriter.Status)
	}
}
