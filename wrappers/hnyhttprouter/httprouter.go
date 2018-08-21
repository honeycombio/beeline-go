package hnyhttprouter

import (
	"net/http"
	"reflect"
	"runtime"

	"github.com/honeycombio/beeline-go/wrappers/common"
	"github.com/julienschmidt/httprouter"
)

// Middleware wraps httprouter handlers. Since it wraps handlers with explicit
// parameters, it can add those values to the event it generates.
func Middleware(handle httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, span := common.StartSpanOrTraceFromHTTP(r)
		defer span.Finish()
		// push the context with our trace and span on to the request
		r = r.WithContext(ctx)

		// replace the writer with our wrapper to catch the status code
		wrappedWriter := common.NewResponseWriter(w)

		// pull out any variables in the URL, add the thing we're matching, etc.
		for _, param := range ps {
			span.AddField("handler.vars."+param.Key, param.Value)
		}
		name := runtime.FuncForPC(reflect.ValueOf(handle).Pointer()).Name()
		span.AddField("handler.name", name)
		span.AddField("name", name)

		handle(w, r, ps)

		if wrappedWriter.Status == 0 {
			wrappedWriter.Status = 200
		}
		span.AddField("response.status_code", wrappedWriter.Status)
	}
}
