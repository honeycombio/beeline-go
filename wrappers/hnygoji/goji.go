package hnygoji

import (
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/internal"
	libhoney "github.com/honeycombio/libhoney-go"
	"goji.io/middleware"
	"goji.io/pat"
)

// Middleware is specifically to use with goji's router.Use() function for
// inserting middleware
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
		// get bits about the handler
		handler := middleware.Handler(ctx)
		if handler == nil {
			ev.AddField("handler.name", "http.NotFound")
			handler = http.NotFoundHandler()
		} else {
			hType := reflect.TypeOf(handler)
			ev.AddField("handler.type", hType.String())
			ev.AddField("handler.name", runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name())
		}
		// find any matched patterns
		pm := middleware.Pattern(ctx)
		if pm != nil {
			// TODO put a regex on `p.String()` to pull out any `:foo` and then
			// use those instead of trying to pull them out of the pattern some
			// other way
			if p, ok := pm.(*pat.Pattern); ok {
				ev.AddField("goji.pat", p.String())
				ev.AddField("goji.methods", p.HTTPMethods())
				ev.AddField("goji.path_prefix", p.PathPrefix())
				patvar := strings.TrimPrefix(p.String(), p.PathPrefix()+":")
				ev.AddField("goji.pat."+patvar, pat.Param(r, patvar))
			} else {
				ev.AddField("pat", "NOT pat.Pattern")

			}
		}
		// TODO get all the parameters and their values
		handler.ServeHTTP(wrappedWriter, r)
		if wrappedWriter.Status == 0 {
			wrappedWriter.Status = 200
		}
		ev.AddField("response.status_code", wrappedWriter.Status)
		ev.AddField("duration_ms", float64(time.Since(start))/float64(time.Millisecond))
	}
	return http.HandlerFunc(wrappedHandler)
}
