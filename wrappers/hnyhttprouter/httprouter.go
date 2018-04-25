package hnyhttprouter

import (
	"net/http"
	"reflect"
	"runtime"
	"time"

	"github.com/honeycombio/honeycomb-go-magic"
	"github.com/honeycombio/honeycomb-go-magic/internal"
	"github.com/honeycombio/libhoney-go"
	"github.com/julienschmidt/httprouter"
)

func Middleware(handle httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		// TODO find out if we're a sub-handler and don't stomp the parent
		// event, or at least get parent/child IDs and intentionally send a
		// subevent or something
		start := time.Now()
		ev := honeycomb.ContextEvent(ctx)
		if ev == nil {
			ev = libhoney.NewEvent()
			defer ev.Send()
			// put the event on the context for everybody downsteam to use
			r = r.WithContext(honeycomb.ContextWithEvent(ctx, ev))
		}
		// pull out any variables in the URL, add the thing we're matching, etc.
		for _, param := range ps {
			ev.AddField("handler.vars."+param.Key, param.Value)
		}
		// add some common fields from the request to our event
		internal.AddRequestProps(r, ev)
		// replace the writer with our wrapper to catch the status code
		wrappedWriter := &internal.ResponseWriter{ResponseWriter: w}
		ev.AddField("handler.name", runtime.FuncForPC(reflect.ValueOf(handle).Pointer()).Name())

		handle(w, r, ps)

		if wrappedWriter.Status == 0 {
			wrappedWriter.Status = 200
		}
		ev.AddField("response.status_code", wrappedWriter.Status)
		ev.AddField("duration_ms", float64(time.Since(start))/float64(time.Millisecond))
	}
}
