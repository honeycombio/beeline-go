package hnynethttp

import (
	"net/http"
	"reflect"
	"runtime"
	"time"

	honeycomb "github.com/honeycombio/honeycomb-go-magic"
	"github.com/honeycombio/honeycomb-go-magic/internal"
	libhoney "github.com/honeycombio/libhoney-go"
)

// WrapHandler will create a Honeycomb event per invocation of this handler with
// all the standard HTTP fields attached.
func WrapHandler(handler http.Handler) http.Handler {
	handlerName := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
	wrappedHandler := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// TODO find out if we're a sub-handler and don't stomp the parent event
		// - get parent/child IDs and intentionally send a subevent
		ev := honeycomb.ContextEvent(r.Context())
		if ev == nil {
			ev = libhoney.NewEvent()
			// put the event on the context for everybody downsteam to use
			r = r.WithContext(honeycomb.ContextWithEvent(r.Context(), ev))
		}
		// add some common fields from the request to our event
		internal.AddRequestProps(r, ev)
		// replace the writer with our wrapper to catch the status code
		wrappedWriter := &internal.ResponseWriter{ResponseWriter: w}
		// add the name of the handler func we're about to invoke
		ev.AddField("handler.name", handlerName)
		handler.ServeHTTP(wrappedWriter, r)
		if wrappedWriter.Status == 0 {
			wrappedWriter.Status = 200
		}
		ev.AddField("response.status_code", wrappedWriter.Status)
		ev.AddField("durationMs", float64(time.Since(start))/float64(time.Millisecond))
		ev.Send()
	}
	return http.HandlerFunc(wrappedHandler)
}

// WrapHandlerFunc will create a Honeycomb event per invocation of this handler
// function with all the standard HTTP fields attached.
func WrapHandlerFunc(hf func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	handlerFuncName := runtime.FuncForPC(reflect.ValueOf(hf).Pointer()).Name()
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ev := honeycomb.ContextEvent(r.Context())
		if ev == nil {
			ev = libhoney.NewEvent()
			// put the event on the context for everybody downstream to use
			r = r.WithContext(honeycomb.ContextWithEvent(r.Context(), ev))
		}
		// add some common fields from the request to our event
		internal.AddRequestProps(r, ev)
		// replace the writer with our wrapper to catch the status code
		wrappedWriter := &internal.ResponseWriter{ResponseWriter: w}
		// add the name of the handler func we're about to invoke
		ev.AddField("handler_func_name", handlerFuncName)

		hf(wrappedWriter, r)
		if wrappedWriter.Status == 0 {
			wrappedWriter.Status = 200
		}
		ev.AddField("response.status_code", wrappedWriter.Status)
		ev.AddField("durationMs", float64(time.Since(start))/float64(time.Millisecond))
		ev.Send()
	}
}

// WrapMuxHandler wraps an http.ServeMux and returns an http.Handler. It is
// intended to be used to wrap a ServeMux when it is passed to
// http.ListenAndServe after all the handlers have been added to the ServeMux.
func WrapMuxHandler(mux *http.ServeMux) http.Handler {
	wrappedHandler := func(w http.ResponseWriter, r *http.Request) {
		// TODO find out if we're a sub-handler and don't stomp the parent
		// event, or at least get parent/child IDs and intentionally send a
		// subevent or something
		start := time.Now()
		ev := honeycomb.ContextEvent(r.Context())
		if ev == nil {
			ev = libhoney.NewEvent()
			// put the event on the context for everybody downstream to use
			r = r.WithContext(honeycomb.ContextWithEvent(r.Context(), ev))
		}
		// add some common fields from the request to our event
		internal.AddRequestProps(r, ev)
		// replace the writer with our wrapper to catch the status code
		wrappedWriter := &internal.ResponseWriter{ResponseWriter: w}
		handler, pat := mux.Handler(r)
		handlerName := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
		hType := reflect.TypeOf(handler).String()
		ev.AddField("mux.handler.pattern", pat)
		ev.AddField("mux.handler.type", hType)
		ev.AddField("mux.handler.name", handlerName)
		handler.ServeHTTP(wrappedWriter, r)
		if wrappedWriter.Status == 0 {
			wrappedWriter.Status = 200
		}
		ev.AddField("response.status_code", wrappedWriter.Status)
		ev.AddField("durationMs", float64(time.Since(start))/float64(time.Millisecond))
		ev.Send()
	}
	return http.HandlerFunc(wrappedHandler)
}
