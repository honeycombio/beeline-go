package honeycomb

import (
	"context"
	"net/http"
	"reflect"
	"runtime"
	"time"

	gorillamux "github.com/gorilla/mux"
	libhoney "github.com/honeycombio/libhoney-go"
)

func AddGorillaMiddleware(handler http.Handler) http.Handler {
	wrappedHandler := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// TODO find out if we're a sub-handler and don't stomp the parent
		// event, or at least get parent/child IDs and intentionally send a
		// subevent or something
		ev := libhoney.NewEvent()
		// put the event on the context for everybody downsteam to use
		r = r.WithContext(context.WithValue(r.Context(), honeyEventContextKey, ev))
		// add some common fields from the request to our event
		addRequestProps(r, ev)
		// replace the writer with our wrapper to catch the status code
		wrappedWriter := &hnyResponseWriter{ResponseWriter: w}
		// pull out any variables in the URL, add the thing we're matching, etc.
		vars := gorillamux.Vars(r)
		for k, v := range vars {
			ev.AddField("vars."+k, v)
		}
		route := gorillamux.CurrentRoute(r)
		chosenHandler := route.GetHandler()
		ev.AddField("chosenHandler_name", runtime.FuncForPC(reflect.ValueOf(chosenHandler).Pointer()).Name())
		if name := route.GetName(); name != "" {
			ev.AddField("handlerName", name)
		}
		if path, err := route.GetPathTemplate(); err == nil {
			ev.AddField("gorilla.routeMatched", path)
		}
		handler.ServeHTTP(wrappedWriter, r)
		if wrappedWriter.status == 0 {
			wrappedWriter.status = 200
		}
		ev.AddField("response.status_code", wrappedWriter.status)
		ev.AddField("duration_ms", float64(time.Since(start))/float64(time.Millisecond))
		ev.Send()
	}
	return http.HandlerFunc(wrappedHandler)
}
