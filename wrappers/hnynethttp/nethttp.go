package hnynethttp

import (
	"context"
	"net/http"
	"reflect"
	"runtime"

	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/internal"
)

// WrapHandler will create a Honeycomb event per invocation of this handler with
// all the standard HTTP fields attached. If passed a ServeMux instead, pull
// what you can from there
func WrapHandler(handler http.Handler) http.Handler {
	// if we can cache handlerName here, let's do so for efficiency's sake
	handlerName := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()

	wrappedHandler := func(w http.ResponseWriter, r *http.Request) {
		var ctx context.Context
		if !beeline.HasTrace(r.Context()) {
			traceHeaders := internal.FindTraceHeaders(r)
			ctx = beeline.StartSpan(r.Context())
			beeline.SetTraceIDs(ctx, traceHeaders.TraceID, traceHeaders.ParentID)
		} else {
			ctx = beeline.StartSpan(r.Context())
		}
		defer beeline.EndSpan(ctx)
		r = r.WithContext(ctx)
		// // TODO find out if we're a sub-handler and don't stomp the parent event
		// // - get parent/child IDs and intentionally send a subevent
		// ev := beeline.ContextEvent(r.Context())
		// if ev == nil {
		// 	ev = libhoney.NewEvent()
		// 	// put the event on the context for everybody downsteam to use
		// 	r = r.WithContext(beeline.ContextWithEvent(r.Context(), ev))
		// }
		// add some common fields from the request to our event
		for k, v := range internal.GetRequestProps(r) {
			internal.AddField(ctx, k, v)
		}
		// replace the writer with our wrapper to catch the status code
		wrappedWriter := internal.NewResponseWriter(w)

		mux, ok := handler.(*http.ServeMux)
		if ok {
			// this is actually a mux! let's do extra muxxy stuff
			handler, pat := mux.Handler(r)
			name := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
			hType := reflect.TypeOf(handler).String()
			internal.AddField(ctx, "handler.pattern", pat)
			internal.AddField(ctx, "handler.type", hType)
			if name != "" {
				internal.AddField(ctx, "handler.name", name)
				internal.AddField(ctx, "name", name)
			}
		} else {
			if handlerName != "" {
				internal.AddField(ctx, "handler.name", handlerName)
				internal.AddField(ctx, "name", handlerName)
			}
		}

		handler.ServeHTTP(wrappedWriter, r)
		if wrappedWriter.Status == 0 {
			wrappedWriter.Status = 200
		}
		internal.AddField(ctx, "response.status_code", wrappedWriter.Status)
		// ev.Metadata, _ = ev.Fields()["name"]
		// ev.Send()
	}
	return http.HandlerFunc(wrappedHandler)
}

// WrapHandlerFunc will create a Honeycomb event per invocation of this handler
// function with all the standard HTTP fields attached.
func WrapHandlerFunc(hf func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	handlerFuncName := runtime.FuncForPC(reflect.ValueOf(hf).Pointer()).Name()
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := beeline.StartSpan(r.Context())
		defer beeline.EndSpan(ctx)
		r = r.WithContext(ctx)
		// add some common fields from the request to our event
		for k, v := range internal.GetRequestProps(r) {
			internal.AddField(ctx, k, v)
		}
		// replace the writer with our wrapper to catch the status code
		wrappedWriter := internal.NewResponseWriter(w)
		// add the name of the handler func we're about to invoke
		if handlerFuncName != "" {
			internal.AddField(ctx, "handler_func_name", handlerFuncName)
			internal.AddField(ctx, "name", handlerFuncName)
		}

		hf(wrappedWriter, r)
		if wrappedWriter.Status == 0 {
			wrappedWriter.Status = 200
		}
		internal.AddField(ctx, "response.status_code", wrappedWriter.Status)
	}
}
