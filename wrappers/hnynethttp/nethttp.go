package hnynethttp

import (
	"net/http"
	"reflect"
	"runtime"
	"time"

	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/internal"
	"github.com/honeycombio/libhoney-go"
)

// WrapHandler will create a Honeycomb event per invocation of this handler with
// all the standard HTTP fields attached. If passed a ServeMux instead, pull
// what you can from there
func WrapHandler(handler http.Handler) http.Handler {
	// if we can cache handlerName here, let's do so for efficiency's sake
	handlerName := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()

	wrappedHandler := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// TODO find out if we're a sub-handler and don't stomp the parent event
		// - get parent/child IDs and intentionally send a subevent
		ev := beeline.ContextEvent(r.Context())
		if ev == nil {
			ev = libhoney.NewEvent()
			// put the event on the context for everybody downsteam to use
			r = r.WithContext(beeline.ContextWithEvent(r.Context(), ev))
		}
		// add some common fields from the request to our event
		internal.AddRequestProps(r, ev)
		// replace the writer with our wrapper to catch the status code
		wrappedWriter := &internal.ResponseWriter{ResponseWriter: w}

		mux, ok := handler.(*http.ServeMux)
		if ok {
			// this is actually a mux! let's do extra muxxy stuff
			handler, pat := mux.Handler(r)
			name := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
			hType := reflect.TypeOf(handler).String()
			ev.AddField("handler.pattern", pat)
			ev.AddField("handler.type", hType)
			if name != "" {
				ev.AddField("handler.name", name)
				ev.AddField("name", name)
			}
		} else {
			if handlerName != "" {
				ev.AddField("handler.name", handlerName)
				ev.AddField("name", handlerName)
			}
		}

		handler.ServeHTTP(wrappedWriter, r)
		if wrappedWriter.Status == 0 {
			wrappedWriter.Status = 200
		}
		ev.AddField("response.status_code", wrappedWriter.Status)
		ev.AddField("duration_ms", float64(time.Since(start))/float64(time.Millisecond))
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
		ev := beeline.ContextEvent(r.Context())
		if ev == nil {
			ev = libhoney.NewEvent()
			// put the event on the context for everybody downstream to use
			r = r.WithContext(beeline.ContextWithEvent(r.Context(), ev))
		}
		// add some common fields from the request to our event
		internal.AddRequestProps(r, ev)
		// replace the writer with our wrapper to catch the status code
		wrappedWriter := &internal.ResponseWriter{ResponseWriter: w}
		// add the name of the handler func we're about to invoke
		if handlerFuncName != "" {
			ev.AddField("handler_func_name", handlerFuncName)
			ev.AddField("name", handlerFuncName)
		}

		hf(wrappedWriter, r)
		if wrappedWriter.Status == 0 {
			wrappedWriter.Status = 200
		}
		ev.AddField("response.status_code", wrappedWriter.Status)
		ev.AddField("duration_ms", float64(time.Since(start))/float64(time.Millisecond))
		ev.Send()
	}
}
