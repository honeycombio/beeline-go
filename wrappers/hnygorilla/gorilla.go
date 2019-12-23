package hnygorilla

import (
	"net/http"
	"reflect"
	"runtime"

	"github.com/gorilla/mux"
	"github.com/honeycombio/beeline-go/propagation"
	"github.com/honeycombio/beeline-go/wrappers/common"
)

// Middleware is a gorilla middleware to add Honeycomb instrumentation to the
// gorilla muxer.
func Middleware(handler http.Handler) http.Handler {
	return MWDelegateHeader(handler, nil)
}

// MWDelegateHeader is a gorilla middleware to add Honeycomb instrumentation to
// the gorilla muxer. The second argument is a function that will examine a
// request and return the appropriate trace header or `nil`. This delegation
// allows the caller to manage custom trace propagation formats as well as
// choose when to trust incoming trace headers.
func MWDelegateHeader(handler http.Handler, fetchTraceHeader func(*http.Request) (*propagation.Propagation, error)) http.Handler {
	wrappedHandler := func(w http.ResponseWriter, r *http.Request) {
		// get a new context with our trace from the request, and add common fields
		ctx, span := common.StartSpanOrTraceFromHTTPDelegateHeader(r, fetchTraceHeader)
		defer span.Send()
		// push the context with our trace and span on to the request
		r = r.WithContext(ctx)

		// replace the writer with our wrapper to catch the status code
		wrappedWriter := common.NewResponseWriter(w)
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
		handler.ServeHTTP(wrappedWriter.Wrapped, r)
		if wrappedWriter.Status == 0 {
			wrappedWriter.Status = 200
		}
		span.AddField("response.status_code", wrappedWriter.Status)
	}
	return http.HandlerFunc(wrappedHandler)
}
