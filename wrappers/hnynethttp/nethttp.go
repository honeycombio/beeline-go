package hnynethttp

import (
	"net/http"
	"reflect"
	"runtime"

	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/internal"
	"github.com/honeycombio/beeline-go/timer"
	libhoney "github.com/honeycombio/libhoney-go"
)

// WrapHandler will create a Honeycomb event per invocation of this handler with
// all the standard HTTP fields attached. If passed a ServeMux instead, pull
// what you can from there
func WrapHandler(handler http.Handler) http.Handler {
	// if we can cache handlerName here, let's do so for efficiency's sake
	handlerName := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()

	wrappedHandler := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var span *internal.Span
		if !beeline.HasTrace(r.Context()) {
			// pick up any trace context from our caller, if present
			traceHeaders, traceContext, _ := internal.FindTraceHeaders(r)
			// use the trace IDs found to spin up a new trace
			ctx, span = internal.StartTraceWithIDs(r.Context(),
				traceHeaders.TraceID, traceHeaders.ParentID, "")
			trace := internal.GetTraceFromContext(ctx)
			// add any additional context to the trace
			for k, v := range traceContext {
				trace.AddField(k, v)
			}
			// and make sure it gets completely sent when we're done.
			defer trace.Send()
		} else {
			// if we're not the root span, just add another layer to our trace.
			ctx, span = internal.StartSpan(r.Context(), "")
		}
		defer span.Finish()
		// push the context with our trace on to the request
		r = r.WithContext(ctx)
		// go get any common HTTP headers and attributes to add to the span
		for k, v := range internal.GetRequestProps(r) {
			span.AddField(k, v)
		}
		// replace the writer with our wrapper to catch the status code
		wrappedWriter := internal.NewResponseWriter(w)

		mux, ok := handler.(*http.ServeMux)
		if ok {
			// this is actually a mux! let's do extra muxxy stuff
			handler, pat := mux.Handler(r)
			name := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
			hType := reflect.TypeOf(handler).String()
			span.AddField("handler.pattern", pat)
			span.AddField("handler.type", hType)
			if name != "" {
				span.AddField("handler.name", name)
				span.AddField("name", name)
			}
		} else {
			if handlerName != "" {
				span.AddField("handler.name", handlerName)
				span.AddField("name", handlerName)
			}
		}

		handler.ServeHTTP(wrappedWriter, r)
		if wrappedWriter.Status == 0 {
			wrappedWriter.Status = 200
		}
		span.AddField("response.status_code", wrappedWriter.Status)
	}
	return http.HandlerFunc(wrappedHandler)
}

// WrapHandlerFunc will create a Honeycomb event per invocation of this handler
// function with all the standard HTTP fields attached.
func WrapHandlerFunc(hf func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	handlerFuncName := runtime.FuncForPC(reflect.ValueOf(hf).Pointer()).Name()
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var span *internal.Span
		if !beeline.HasTrace(r.Context()) {
			// pick up any trace context from our caller, if present
			traceHeaders, traceContext, _ := internal.FindTraceHeaders(r)
			// use the trace IDs found to spin up a new trace
			ctx, span = internal.StartTraceWithIDs(r.Context(),
				traceHeaders.TraceID, traceHeaders.ParentID, "")
			trace := internal.GetTraceFromContext(ctx)
			// add any additional context to the trace
			for k, v := range traceContext {
				trace.AddField(k, v)
			}
			// and make sure it gets completely sent when we're done.
			defer trace.Send()
		} else {
			// if we're not the root span, just add another layer to our trace.
			ctx, span = internal.StartSpan(r.Context(), "")
		}
		defer span.Finish()
		// push the context with our trace on to the request
		r = r.WithContext(ctx)
		// add some common fields from the request to our event
		for k, v := range internal.GetRequestProps(r) {
			span.AddField(k, v)
		}
		// replace the writer with our wrapper to catch the status code
		wrappedWriter := internal.NewResponseWriter(w)
		// add the name of the handler func we're about to invoke
		if handlerFuncName != "" {
			span.AddField("handler_func_name", handlerFuncName)
			span.AddField("name", handlerFuncName)
		}

		hf(wrappedWriter, r)
		if wrappedWriter.Status == 0 {
			wrappedWriter.Status = 200
		}
		span.AddField("response.status_code", wrappedWriter.Status)
	}
}

type hnyTripper struct {
	// wrt is the wrapped round tripper
	wrt http.RoundTripper
}

func (ht *hnyTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	ctx := r.Context()
	// if there's no trace in the context, just send an event
	if !beeline.HasTrace(ctx) {
		tm := timer.Start()
		ev := libhoney.NewEvent()
		defer ev.Send()

		ev.AddField("meta.type", "http_client")

		resp, err := ht.wrt.RoundTrip(r)

		if err != nil {
			ev.AddField("error", err.Error())
		}
		dur := tm.Finish()
		ev.AddField("duration_ms", dur)
		return resp, err
	}
	// we have a trace, let's use it and pass along trace context in addition to
	// making a span around this HTTP call
	var span *internal.Span
	ctx, span = internal.StartSpan(ctx, "http_client")
	defer span.Finish()
	r = r.WithContext(ctx)
	span.AddField("meta.type", "http_client")
	r.Header.Add(internal.TracePropagationHTTPHeader, internal.MarshalTraceContext(ctx))

	// add in common request headers.
	reqprops := internal.GetRequestProps(r)
	for k, v := range reqprops {
		span.AddField(k, v)
	}

	resp, err := ht.wrt.RoundTrip(r)

	if err != nil {
		span.AddField("error", err.Error())
	} else {
		span.AddField("resp.status_code", resp.StatusCode)

	}
	return resp, err
}

// WrapRoundTripper wraps an http transport for outgoing HTTP calls. Using a
// wrapped transport will send an event to Honeycomb for each outbound HTTP call
// you make. Include a context with outbound requests when possible to enable
// correlation
func WrapRoundTripper(r http.RoundTripper) http.RoundTripper {
	return &hnyTripper{
		wrt: r,
	}
}
