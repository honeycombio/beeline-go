package honeycomb

import (
	"context"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"time"

	libhoney "github.com/honeycombio/libhoney-go"
	"goji.io/middleware"
	"goji.io/pat"
)

// InstrumentGojiMiddleware is specifically to use with goji's router.Use()
// function for inserting middleware
func InstrumentGojiMiddleware(handler http.Handler) http.Handler {
	wrappedHandler := func(w http.ResponseWriter, r *http.Request) {
		// TODO find out if we're a sub-handler and don't stomp the parent
		// event, or at least get parent/child IDs and intentionally send a
		// subevent or something
		start := time.Now()
		ev := libhoney.NewEvent()
		// put the event on the context for everybody downsteam to use
		r = r.WithContext(context.WithValue(r.Context(), honeyEventContextKey, ev))
		ctx := r.Context()
		// add some common fields from the request to our event
		addRequestProps(r, ev)
		// replace the writer with our wrapper to catch the status code
		wrappedWriter := &hnyResponseWriter{ResponseWriter: w}
		// get bits about the handler
		handler := middleware.Handler(ctx)
		if handler == nil {
			ev.AddField("handler_name", "http.NotFound")
			handler = http.NotFoundHandler()
		} else {
			hType := reflect.TypeOf(handler)
			ev.AddField("handlerType", hType.String())
			ev.AddField("handler_name", runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name())
		}
		// find any matched patterns
		pm := middleware.Pattern(ctx)
		if pm != nil {
			// TODO put a regex on `p.String()` to pull out any `:foo` and then
			// use those instead of trying to pull them out of the pattern some
			// other way
			if p, ok := pm.(*pat.Pattern); ok {
				ev.AddField("pat", "is a pat.Pattern")
				ev.AddField("goji_pat", p.String())
				ev.AddField("goji_methods", p.HTTPMethods())
				ev.AddField("goji_path_prefix", p.PathPrefix())
				patvar := strings.TrimPrefix(p.String(), p.PathPrefix()+":")
				ev.AddField("patvar", patvar)
				// ev.AddField("patval", pat.Param(r, patvar))
			} else {
				ev.AddField("pat", "NOT pat.Pattern")

			}
		}
		// TODO get all the parameters and their values
		handler.ServeHTTP(wrappedWriter, r)
		if wrappedWriter.status == 0 {
			wrappedWriter.status = 200
		}
		ev.AddField("response.status_code", wrappedWriter.status)
		ev.AddField("durationMs", float64(time.Since(start))/float64(time.Millisecond))
		ev.Send()
	}
	return http.HandlerFunc(wrappedHandler)
}
