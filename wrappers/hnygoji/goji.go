package hnygoji

import (
	"net/http"
	"reflect"
	"runtime"
	"strings"

	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/internal"
	"goji.io/middleware"
	"goji.io/pat"
)

// Middleware is specifically to use with goji's router.Use() function for
// inserting middleware
func Middleware(handler http.Handler) http.Handler {
	wrappedHandler := func(w http.ResponseWriter, r *http.Request) {
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
		// push the context with our new span and trace on to the request
		r = r.WithContext(ctx)
		// go get any common HTTP headers and attributes to add to the span
		for k, v := range internal.GetRequestProps(r) {
			span.AddField(k, v)
		}
		// replace the writer with our wrapper to catch the status code
		wrappedWriter := internal.NewResponseWriter(w)

		// get bits about the handler
		handler := middleware.Handler(ctx)
		if handler == nil {
			span.AddField("handler.name", "http.NotFound")
			handler = http.NotFoundHandler()
		} else {
			hType := reflect.TypeOf(handler)
			span.AddField("handler.type", hType.String())
			name := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
			span.AddField("handler.name", name)
			span.AddField("name", name)
		}
		// find any matched patterns
		pm := middleware.Pattern(ctx)
		if pm != nil {
			// TODO put a regex on `p.String()` to pull out any `:foo` and then
			// use those instead of trying to pull them out of the pattern some
			// other way
			if p, ok := pm.(*pat.Pattern); ok {
				span.AddField("goji.pat", p.String())
				span.AddField("goji.methods", p.HTTPMethods())
				span.AddField("goji.path_prefix", p.PathPrefix())
				patvar := strings.TrimPrefix(p.String(), p.PathPrefix()+":")
				span.AddField("goji.pat."+patvar, pat.Param(r, patvar))
			} else {
				span.AddField("pat", "NOT pat.Pattern")

			}
		}
		// TODO get all the parameters and their values
		handler.ServeHTTP(wrappedWriter, r)
		if wrappedWriter.Status == 0 {
			wrappedWriter.Status = 200
		}
		span.AddField("response.status_code", wrappedWriter.Status)
	}
	return http.HandlerFunc(wrappedHandler)
}
