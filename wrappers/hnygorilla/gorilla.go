package hnygorilla

import (
	"net/http"
	"reflect"
	"runtime"
	"time"

	"github.com/gorilla/mux"
	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/internal"
	"github.com/honeycombio/libhoney-go"
)

// Middleware is a gorilla middleware to add Honeycomb instrumentation to the
// gorilla muxer.
func Middleware(handler http.Handler) http.Handler {
	wrappedHandler := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// TODO find out if we're a sub-handler and don't stomp the parent
		// event, or at least get parent/child IDs and intentionally send a
		// subevent or something
		start := time.Now()
		ev := beeline.ContextEvent(ctx)
		if ev == nil {
			ev = libhoney.NewEvent()
			defer ev.Send()
			// put the event on the context for everybody downsteam to use
			r = r.WithContext(beeline.ContextWithEvent(ctx, ev))
		}
		// add some common fields from the request to our event
		internal.AddRequestProps(r, ev)
		// replace the writer with our wrapper to catch the status code
		wrappedWriter := &internal.ResponseWriter{ResponseWriter: w}
		// pull out any variables in the URL, add the thing we're matching, etc.
		vars := mux.Vars(r)
		for k, v := range vars {
			ev.AddField("gorilla.vars."+k, v)
		}
		route := mux.CurrentRoute(r)
		chosenHandler := route.GetHandler()
		ev.AddField("handler.fnname", runtime.FuncForPC(reflect.ValueOf(chosenHandler).Pointer()).Name())
		if name := route.GetName(); name != "" {
			ev.AddField("handler.name", name)
		}
		if path, err := route.GetPathTemplate(); err == nil {
			ev.AddField("handler.route", path)
		}
		handler.ServeHTTP(wrappedWriter, r)
		if wrappedWriter.Status == 0 {
			wrappedWriter.Status = 200
		}
		ev.AddField("response.status_code", wrappedWriter.Status)
		ev.AddField("duration_ms", float64(time.Since(start))/float64(time.Millisecond))
	}
	return http.HandlerFunc(wrappedHandler)
}
