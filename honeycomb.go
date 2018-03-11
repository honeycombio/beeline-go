package honeycomb

import (
	"context"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"

	libhoney "github.com/honeycombio/libhoney-go"
	"goji.io/middleware"
	"goji.io/pat"
)

const honeyEventContextKey = "honeycombEventContextKey"

type Hny struct {
}

type hnyResponseWriter struct {
	http.ResponseWriter
	status int
}

func (h *hnyResponseWriter) WriteHeader(statusCode int) {
	h.status = statusCode
	h.ResponseWriter.WriteHeader(statusCode)
}

func NewInstrumenter(wk string) *Hny {
	config := libhoney.Config{
		WriteKey: wk,
		Dataset:  "vanilla",
		Output:   &libhoney.WriterOutput{},
	}
	libhoney.Init(config)

	if hostname, err := os.Hostname(); err == nil {
		libhoney.AddField("host", hostname)
	}
	return &Hny{}
}

func AddField(ctx context.Context, key string, val interface{}) {
	ev := existingEventFromContext(ctx)
	if ev == nil {
		return
	}
	ev.AddField(key, val)
}

// HandleFunc func(w, r)
// Handler interface ServeHTTP()
// Mux

func (h *Hny) InstrumentHandleFunc(hf func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO find out if we're a sub-handler and don't stomp the parent
		// event, or at least get parent/child IDs and intentionally send a
		// subevent or something
		start := time.Now()
		ev := existingEventFromContext(r.Context())
		if ev == nil {
			ev = libhoney.NewEvent()
		}
		// put the event on the context for everybody downsteam to use
		r = r.WithContext(context.WithValue(r.Context(), honeyEventContextKey, ev))
		// add some common fields from the request to our event
		addRequestProps(r, ev)
		// replace the writer with our wrapper to catch the status code
		wrappedWriter := &hnyResponseWriter{ResponseWriter: w}
		// add the name of the handler func we're about to invoke
		ev.AddField("handler_func_name", runtime.FuncForPC(reflect.ValueOf(hf).Pointer()).Name())

		hf(wrappedWriter, r)
		if wrappedWriter.status == 0 {
			wrappedWriter.status = 200
		}
		ev.AddField("response.status_code", wrappedWriter.status)
		ev.AddField("duration_ms", float64(time.Since(start))/float64(time.Millisecond))
		ev.Send()
	}
}

func (h *Hny) InstrumentHandler(handler http.Handler) http.Handler {
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
		// add the name of the handler func we're about to invoke
		ev.AddField("handler_name", runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name())
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

func (h *Hny) InstrumentMuxHandler(mux *http.ServeMux) http.Handler {
	wrappedHandler := func(w http.ResponseWriter, r *http.Request) {
		// TODO find out if we're a sub-handler and don't stomp the parent
		// event, or at least get parent/child IDs and intentionally send a
		// subevent or something
		start := time.Now()
		ev := libhoney.NewEvent()
		// put the event on the context for everybody downsteam to use
		r = r.WithContext(context.WithValue(r.Context(), honeyEventContextKey, ev))
		// add some common fields from the request to our event
		addRequestProps(r, ev)
		// replace the writer with our wrapper to catch the status code
		wrappedWriter := &hnyResponseWriter{ResponseWriter: w}
		handler, pat := mux.Handler(r)
		ev.AddField("handlerPattern", pat)
		// get Handler type and name
		hType := reflect.TypeOf(handler)
		ev.AddField("handlerType", hType.String())
		ev.AddField("handler_name", runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name())
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

func (h *Hny) InstrumentGojiMiddleware(handler http.Handler) http.Handler {
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
		if pm == nil {
			ev.AddField("goji_pat", "nil")
		} else {
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
		ev.AddField("duration_ms", float64(time.Since(start))/float64(time.Millisecond))
		ev.Send()
	}
	return http.HandlerFunc(wrappedHandler)
}

// func A(inner http.Handler) http.Handler {
// 	log.Print("A: called")
// 	mw := func(w http.ResponseWriter, r *http.Request) {
// 		log.Print("A: before")
// 		inner.ServeHTTP(w, r)
// 		log.Print("A: after")
// 	}
// 	return http.HandlerFunc(mw)
// }

func addRequestProps(req *http.Request, ev *libhoney.Event) {
	// Add a variety of details about the HTTP request, such as user agent
	// and method, to any created libhoney event.
	ev.AddField("request.method", req.Method)
	ev.AddField("request.path", req.URL.Path)
	ev.AddField("request.host", req.URL.Host)
	ev.AddField("request.proto", req.Proto)
	ev.AddField("request.content_length", req.ContentLength)
	ev.AddField("request.remote_addr", req.RemoteAddr)
	ev.AddField("request.user_agent", req.UserAgent())

}

// parse tracing headers if they exist X-Amzn-Trace-Id
// X-Amzn-Trace-Id: Self=1-67891234-12456789abcdef012345678;Root=1-67891233-abcdef012345678912345678;CalledFrom=app
func parseAWSTraceHeader(req *http.Request, ev *libhoney.Event) {
	traceHeader := req.Header.Get("X-Amzn-Trace-Id")
	if traceHeader == "" {
		// no header found
		return
	}
	// break into key=val pairs on `;` and add each key=val header
	ids := strings.Split(traceHeader, ";")
	for _, id := range ids {
		keyval := strings.Split(id, "=")
		if len(keyval) != 2 {
			// malformed keyval
			continue
		}
		ev.AddField("request.trace_id."+keyval[0], keyval[1])
	}
}

// func InstrumentHandleFunc(pattern string, handler func(ResponseWriter, *Request)) (pattern string, handler func(ResponseWriter, *Request)) {

// }
// func Instrument()

// const honeyBuilderContextKey = "honeycombBuilderContextKey"
// const honeyEventContextKey = "honeycombEventContextKey"
// const honeyGrabBagContextKey = "honeycombGrabBagContextKey"

// // HoneycombHandler will wrap any handler and provide an event in the context
// // for all subsequent handlers. It will also add some instrumentation about that
// // event.
// func HoneycombHandler(h http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		ctx := r.Context()
// 		t := timer{}
// 		t.Start()
// 		log.Println("Before")
// 		h.ServeHTTP(w, r) // call original
// 		log.Println("After")
// 	})
// }

// // ctx = context.WithValue(ctx, builderContextKey, a.Libhoney.NewBuilder())

// func addBuilderToContext(ctx context.Context) context.Context {
// 	return context.WithValue(ctx, honeyBuilderContextKey, newBuilderFromContext(ctx))
// }

// func addEventToContext(ctx context.Context) context.Context {
// 	return context.WithValue(ctx, honeyEventContextKey, newEventFromContext(ctx))
// }

// // might return a thing or nil if there wasn't one there already
// func existingBuilderFromContext(ctx context.Context) *libhoney.Builder {
// 	if bldr, ok := ctx.Value(honeyBuilderContextKey).(*libhoney.Builder); ok {
// 		return bldr
// 	}
// 	return nil
// }

// // creates a copy of what's in the context and returns that
// func newBuilderFromContext(ctx context.Context) *libhoney.Builder {
// 	bldr := existingBuilderFromContext(ctx)
// 	if bldr == nil {
// 		return libhoney.NewBuilder()
// 	}
// 	return bldr.Clone()
// }

// might return a thing or nil if there wasn't one there already
func existingEventFromContext(ctx context.Context) *libhoney.Event {
	if evt, ok := ctx.Value(honeyEventContextKey).(*libhoney.Event); ok {
		return evt
	}
	return nil
}

// // creates a new event from the builder in the context and returns that
// func newEventFromContext(ctx context.Context) *libhoney.Event {
// 	return newBuilderFromContext(ctx).NewEvent()
// }

// // might return a thing or nil if there wasn't one there already
// func existingGrabBagFromContext(ctx context.Context) map[string]interface{} {
// 	if m, ok := ctx.Value(honeyGrabBagContextKey).(map[string]interface{}); ok {
// 		return m
// 	}
// 	return nil
// }

// // creates a shallow copy of what's in the context and returns that
// func newGrabBagFromContext(ctx context.Context) map[string]interface{} {
// 	if m, ok := ctx.Value(honeyGrabBagContextKey).(map[string]interface{}); ok {
// 		newM := make(map[string]interface{})
// 		for k, v := range m {
// 			newM[k] = v
// 		}
// 		return newM
// 	}
// 	return make(map[string]interface{})
// }

// type Timer interface {
// 	Start()
// 	End()
// 	Duration() time.Duration
// }

// type timer struct {
// 	start time.Time
// 	dur   time.Duration
// }

// func (t *timer) Start() {
// 	t.start = time.Now()
// }
// func (t *timer) End() {
// 	t.dur = time.Since(t.start)
// }
// func (t *timer) Duration() time.Duration {
// 	return t.dur
// }
