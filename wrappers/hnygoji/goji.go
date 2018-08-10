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
		if !beeline.HasTrace(r.Context()) {
			// pick up any trace context from our caller, if present
			traceHeaders, traceContext, _ := internal.FindTraceHeaders(r)
			// use the trace IDs found to spin up a new trace
			ctx = beeline.StartTraceWithIDs(r.Context(),
				traceHeaders.TraceID, traceHeaders.ParentID, "")
			trace := internal.GetTraceFromContext(ctx)
			// push the context with our trace on to the request
			r = r.WithContext(ctx)
			// add any additional context to the trace
			for k, v := range traceContext {
				trace.AddField(k, v)
			}
			// and make sure it gets completely sent when we're done.
			defer internal.SendTrace(trace)
		} else {
			// if we're not the root span, just add another layer to our trace.
			internal.PushSpanOnStack(r.Context(), "")
		}
		defer internal.EndSpan(ctx)
		// go get any common HTTP headers and attributes to add to the span
		for k, v := range internal.GetRequestProps(r) {
			internal.AddField(ctx, k, v)
		}
		// replace the writer with our wrapper to catch the status code
		wrappedWriter := internal.NewResponseWriter(w)

		// get bits about the handler
		handler := middleware.Handler(ctx)
		if handler == nil {
			internal.AddField(ctx, "handler.name", "http.NotFound")
			handler = http.NotFoundHandler()
		} else {
			hType := reflect.TypeOf(handler)
			internal.AddField(ctx, "handler.type", hType.String())
			name := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
			internal.AddField(ctx, "handler.name", name)
			internal.AddField(ctx, "name", name)
		}
		// find any matched patterns
		pm := middleware.Pattern(ctx)
		if pm != nil {
			// TODO put a regex on `p.String()` to pull out any `:foo` and then
			// use those instead of trying to pull them out of the pattern some
			// other way
			if p, ok := pm.(*pat.Pattern); ok {
				internal.AddField(ctx, "goji.pat", p.String())
				internal.AddField(ctx, "goji.methods", p.HTTPMethods())
				internal.AddField(ctx, "goji.path_prefix", p.PathPrefix())
				patvar := strings.TrimPrefix(p.String(), p.PathPrefix()+":")
				internal.AddField(ctx, "goji.pat."+patvar, pat.Param(r, patvar))
			} else {
				internal.AddField(ctx, "pat", "NOT pat.Pattern")

			}
		}
		// TODO get all the parameters and their values
		handler.ServeHTTP(wrappedWriter, r)
		if wrappedWriter.Status == 0 {
			wrappedWriter.Status = 200
		}
		internal.AddField(ctx, "response.status_code", wrappedWriter.Status)
	}
	return http.HandlerFunc(wrappedHandler)
}
