package hnynethttp

import (
	"context"
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
		var ctx context.Context
		if !beeline.HasTrace(r.Context()) {
			// pick up any trace context from our caller, if present
			traceHeaders, traceContext, _ := internal.FindTraceHeaders(r)
			// use the trace IDs found to spin up a new trace
			ctx = beeline.StartTraceWithIDs(r.Context(),
				traceHeaders.TraceID, traceHeaders.ParentID, "")
			trace := internal.GetTraceFromContext(ctx)
			// push the context with our trace on to the request
			r = r.WithContext(ctx)
			// add any additional context to the trace
			for k, v := range traceContext {
				trace.AddField(k, v)
			}
			// and make sure it gets completely sent when we're done.
			defer internal.SendTrace(trace)
		} else {
			// if we're not the root span, just add another layer to our trace.
			internal.PushSpanOnStack(r.Context(), "")
		}
		defer internal.EndSpan(ctx)
		// go get any common HTTP headers and attributes to add to the span
		for k, v := range internal.GetRequestProps(r) {
			internal.AddField(ctx, k, v)
		}
		// replace the writer with our wrapper to catch the status code
		wrappedWriter := internal.NewResponseWriter(w)

		mux, ok := handler.(*http.ServeMux)
		if ok {
			// this is actually a mux! let's do extra muxxy stuff
			handler, pat := mux.Handler(r)
			name := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
			hType := reflect.TypeOf(handler).String()
			internal.AddField(ctx, "handler.pattern", pat)
			internal.AddField(ctx, "handler.type", hType)
			if name != "" {
				internal.AddField(ctx, "handler.name", name)
				internal.AddField(ctx, "name", name)
			}
		} else {
			if handlerName != "" {
				internal.AddField(ctx, "handler.name", handlerName)
				internal.AddField(ctx, "name", handlerName)
			}
		}

		handler.ServeHTTP(wrappedWriter, r)
		if wrappedWriter.Status == 0 {
			wrappedWriter.Status = 200
		}
		internal.AddField(ctx, "response.status_code", wrappedWriter.Status)
	}
	return http.HandlerFunc(wrappedHandler)
}

// WrapHandlerFunc will create a Honeycomb event per invocation of this handler
// function with all the standard HTTP fields attached.
func WrapHandlerFunc(hf func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	handlerFuncName := runtime.FuncForPC(reflect.ValueOf(hf).Pointer()).Name()
	return func(w http.ResponseWriter, r *http.Request) {
		var ctx context.Context
		if !beeline.HasTrace(r.Context()) {
			// pick up any trace context from our caller, if present
			traceHeaders, traceContext, _ := internal.FindTraceHeaders(r)
			// use the trace IDs found to spin up a new trace
			ctx = beeline.StartTraceWithIDs(r.Context(),
				traceHeaders.TraceID, traceHeaders.ParentID, "")
			trace := internal.GetTraceFromContext(ctx)
			// push the context with our trace on to the request
			r = r.WithContext(ctx)
			// add any additional context to the trace
			for k, v := range traceContext {
				trace.AddField(k, v)
			}
			// and make sure it gets completely sent when we're done.
			defer internal.SendTrace(trace)
		} else {
			// if we're not the root span, just add another layer to our trace.
			internal.PushSpanOnStack(r.Context(), "")
		}
		defer internal.EndSpan(ctx)
		// add some common fields from the request to our event
		for k, v := range internal.GetRequestProps(r) {
			internal.AddField(ctx, k, v)
		}
		// replace the writer with our wrapper to catch the status code
		wrappedWriter := internal.NewResponseWriter(w)
		// add the name of the handler func we're about to invoke
		if handlerFuncName != "" {
			internal.AddField(ctx, "handler_func_name", handlerFuncName)
			internal.AddField(ctx, "name", handlerFuncName)
		}

		hf(wrappedWriter, r)
		if wrappedWriter.Status == 0 {
			wrappedWriter.Status = 200
		}
		internal.AddField(ctx, "response.status_code", wrappedWriter.Status)
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
	beeline.StartSpan(ctx, "http_client")
	defer beeline.EndSpan(ctx) // TODO concurrency
	internal.AddField(ctx, "meta.type", "http_client")
	r.Header.Add(internal.TracePropagationHTTPHeader, internal.MarshalTraceContext(ctx))

	// add in common request headers.
	reqprops := internal.GetRequestProps(r)
	for k, v := range reqprops {
		internal.AddField(ctx, k, v)
	}

	resp, err := ht.wrt.RoundTrip(r)

	if err != nil {
		internal.AddField(ctx, "error", err.Error())
	} else {
		internal.AddField(ctx, "resp.status_code", resp.StatusCode)

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
