package honeycomb

import (
	"context"
	"net/http"
	"reflect"
	"runtime"

	libhoney "github.com/honeycombio/libhoney-go"
	"github.com/julienschmidt/httprouter"
)

func InstrumentHTTPRouterMiddleware(handle httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		// TODO find out if we're a sub-handler and don't stomp the parent
		// event, or at least get parent/child IDs and intentionally send a
		// subevent or something
		ev := existingEventFromContext(r.Context())
		if ev == nil {
			ev = libhoney.NewEvent()
			// put the event on the context for everybody downsteam to use
			r = r.WithContext(context.WithValue(r.Context(), honeyEventContextKey, ev))
		}
		// pull out any variables in the URL, add the thing we're matching, etc.
		for _, param := range ps {
			ev.AddField("vars."+param.Key, param.Value)
		}
		// spew.Dump(ps)

		ev.AddField("chosenHandle_name", runtime.FuncForPC(reflect.ValueOf(handle).Pointer()).Name())
		handle(w, r, ps)
	}
}
