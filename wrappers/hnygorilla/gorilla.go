package hnygorilla

import (
	"net/http"
	"reflect"
	"runtime"

	"github.com/gorilla/mux"
	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/internal"
)

// Middleware is a gorilla middleware to add Honeycomb instrumentation to the
// gorilla muxer.
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
		defer span.Finish()
		// push the context with our trace on to the request
		r = r.WithContext(ctx)
		// go get any common HTTP headers and attributes to add to the span
		for k, v := range internal.GetRequestProps(r) {
			span.AddField(k, v)
		}
		// replace the writer with our wrapper to catch the status code
		wrappedWriter := internal.NewResponseWriter(w)
		// pull out any variables in the URL, add the thing we're matching, etc.
		vars := mux.Vars(r)
		for k, v := range vars {
			span.AddField("gorilla.vars."+k, v)
		}
		route := mux.CurrentRoute(r)
		if route != nil {
			chosenHandler := route.GetHandler()
			reflectHandler := reflect.ValueOf(chosenHandler)
			if reflectHandler.Kind() == reflect.Func {
				funcName := runtime.FuncForPC(reflectHandler.Pointer()).Name()
				span.AddField("handler.fnname", funcName)
				if funcName != "" {
					span.AddField("name", funcName)
				}
			}
			typeOfHandler := reflect.TypeOf(chosenHandler)
			if typeOfHandler.Kind() == reflect.Struct {
				structName := typeOfHandler.Name()
				if structName != "" {
					span.AddField("name", structName)
				}
			}
			name := route.GetName()
			if name != "" {
				span.AddField("handler.name", name)
				// stomp name because user-supplied names are better than function names
				span.AddField("name", name)
			}
			if path, err := route.GetPathTemplate(); err == nil {
				span.AddField("handler.route", path)
			}
		}
		handler.ServeHTTP(wrappedWriter, r)
		if wrappedWriter.Status == 0 {
			wrappedWriter.Status = 200
		}
		span.AddField("response.status_code", wrappedWriter.Status)
	}
	return http.HandlerFunc(wrappedHandler)
}
