package propagation

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/trace"
	"google.golang.org/grpc/codes"
)

// MarshalHoneycombTraceContext uses the information in prop to create trace context headers
// that conform to the W3C Trace Context specification. The header values are set in headers,
// which is an HTTPSupplier, an interface to which http.Header is an implementation. The headers
// are also returned as a map[string]string.
func MarshalW3CTraceContext(ctx context.Context, prop *PropagationContext) (context.Context, map[string]string) {
	headerMap := make(map[string]string)
	otelSpan, err := createOpenTelemetrySpan(prop)
	if err != nil {
		return ctx, headerMap
	}
	ctx = trace.ContextWithSpan(ctx, otelSpan)
	propagator := trace.DefaultHTTPPropagator()
	supplier := newSupplier()
	propagator.Inject(ctx, supplier)
	for _, key := range propagator.GetAllKeys() {
		headerMap[key] = supplier.Get(key)
	}
	return ctx, headerMap
}

// UnmarshalW3CTraceContext parses the information provided in the appropriate headers
// and creates a PropagationContext instance. Headers are passed in via an HTTPSupplier,
// which is an interface that defines Get and Set methods, http.Header is an implementation.
func UnmarshalW3CTraceContext(ctx context.Context, headers map[string]string) (context.Context, *PropagationContext) {
	supplier := newSupplier()
	for k, v := range headers {
		supplier.Set(k, v)
	}
	propagator := trace.DefaultHTTPPropagator()
	ctx = propagator.Extract(ctx, supplier)
	spanContext := trace.RemoteSpanContextFromContext(ctx)
	return ctx, &PropagationContext{
		TraceID:    spanContext.TraceID.String(),
		ParentID:   spanContext.SpanID.String(),
		TraceFlags: spanContext.TraceFlags,
	}
}

// createOpenTelemetrySpan creates a shell trace.Span with information from the provided
// PropagationContext. It's a shell because the only field populated is the span context.
func createOpenTelemetrySpan(prop *PropagationContext) (trace.Span, error) {
	if prop == nil {
		return otelSpan{}, nil
	}

	traceID, err := trace.IDFromHex(prop.TraceID)
	if err != nil {
		return nil, err
	}
	spanID, err := trace.SpanIDFromHex(prop.ParentID)
	if err != nil {
		return nil, err
	}

	spanCtx := trace.SpanContext{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: prop.TraceFlags,
	}

	return otelSpan{
		ctx: spanCtx,
	}, nil
}

// otelSpan is an implementation of the trace.Span interface. That interface is fairly
// wide, so there are a lot of methods on this type that are noops. The only field we
// need in order to use the opentelemetry-go sdk w3c trace context propagator is the
// trace.SpanContext, so we populate that.
type otelSpan struct {
	ctx trace.SpanContext
}

// SpanContext returns the trace.SpanContext, which is the only field expected to exist.
func (os otelSpan) SpanContext() trace.SpanContext {
	return os.ctx
}

// IsRecording returns false. It exists to satisfy the trace.Span interface.
func (os otelSpan) IsRecording() bool {
	return false
}

// SetStatus does nothing. It exists to satisfy the trace.Span interface.
func (os otelSpan) SetStatus(code codes.Code, msg string) {
	return
}

// SetAttribute does nothing. It exists to satisfy the trace.Span interface.
func (os otelSpan) SetAttribute(k string, v interface{}) {
	return
}

// SetAttributes does nothing. It exists to satisfy the trace.Span interface.
func (os otelSpan) SetAttributes(attributes ...kv.KeyValue) {
	return
}

// End does nothing. It exists to satisfy the trace.Span interface.
func (os otelSpan) End(options ...trace.EndOption) {
	return
}

// RecordError does nothing. It exists to satisfy the trace.Span interface.
func (os otelSpan) RecordError(ctx context.Context, err error, opts ...trace.ErrorOption) {
	return
}

// Tracer returns nil. It exists to satisfy the trace.Span interface.
func (os otelSpan) Tracer() trace.Tracer {
	return nil
}

// AddEvent does nothing. It exists to satisfy the trace.Span interface.
func (os otelSpan) AddEvent(ctx context.Context, name string, attrs ...kv.KeyValue) {
	return
}

// AddEventWithTimestamp does nothing. It exists to satisfy the trace.Span interface.
func (os otelSpan) AddEventWithTimestamp(ctx context.Context, timestamp time.Time, name string, attrs ...kv.KeyValue) {
	return
}

// SetName does nothing. It exists to satisfy the trace.Span interface.
func (os otelSpan) SetName(name string) {
	return
}

// supplier is a container for values, which is a map of strings to strings. It is intended to
// hold http headers used by the OpenTelemetry SDK. It exists to satisfy the method signatures
// for the opentelemetry sdk but is not part of the beeline trace API.
type supplier struct {
	values map[string]string
}

// newSupplier creates and returns an empty supplier with an initialized map for values.
func newSupplier() *supplier {
	m := &supplier{}
	m.values = make(map[string]string)
	return m
}

// Get returns the value associated with the provided key, if any.
func (m supplier) Get(key string) string {
	if value, ok := m.values[key]; ok {
		return value
	}
	return ""
}

// Set associates the provided value with the provided key.
func (m supplier) Set(key string, value string) {
	m.values[key] = value
}
