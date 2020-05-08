package trace

import (
	"context"
	"encoding/hex"
	"time"

	"go.opentelemetry.io/otel/api/kv"
	oteltrace "go.opentelemetry.io/otel/api/trace"
	"google.golang.org/grpc/codes"
)

// otelSpan implements otel/api/trace.Span. Though it defines methods to satisfy
// that interface, none of them are implemented except for SpanContext. This is
// used for compatibilty between beelines and OTel-Go.
type otelSpan struct {
	spanContext oteltrace.SpanContext
}

func (os otelSpan) SpanContext() oteltrace.SpanContext {
	return os.spanContext
}

func (os otelSpan) IsRecording() bool {
	return false
}

func (os otelSpan) SetStatus(code codes.Code, msg string) {
	return
}

func (os otelSpan) SetAttribute(k string, v interface{}) {
	return
}

func (os otelSpan) SetAttributes(attributes ...kv.KeyValue) {
	return
}

func (os otelSpan) End(options ...oteltrace.EndOption) {
	return
}

func (os otelSpan) RecordError(ctx context.Context, err error, opts ...oteltrace.ErrorOption) {
	return
}

func (os otelSpan) Tracer() oteltrace.Tracer {
	return nil
}

func (os otelSpan) AddEvent(ctx context.Context, name string, attrs ...kv.KeyValue) {
	return
}

func (os otelSpan) AddEventWithTimestamp(ctx context.Context, timestamp time.Time, name string, attrs ...kv.KeyValue) {
	return
}

func (os otelSpan) SetName(name string) {
	return
}

// oTelRemoteSpanFromContext looks for an Otel SpanContext in the provided context.
// If it finds a non-empty context, it returns it, otherwise it returns nil.
func oTelRemoteSpanFromContext(ctx context.Context) *oteltrace.SpanContext {
	spanContext := oteltrace.RemoteSpanContextFromContext(ctx)
	if spanContext == oteltrace.EmptySpanContext() {
		return nil
	}
	return &spanContext
}

// OTelSpanFromContext uses the Honeycomb trace and span in the provided
// context to construct an OpenTelemetry SpanContext and returns it wrapped in
// an OTel span. This allows the beeline to use certain tracing and propagation
// methods defined in the OpenTelemetry Go SDK.
func OTelSpanFromContext(ctx context.Context) oteltrace.Span {
	sp := GetSpanFromContext(ctx)

	if sp == nil {
		return otelSpan{}
	}

	traceID, _ := oteltrace.IDFromHex(sp.GetTrace().GetTraceID())
	spanID, _ := oteltrace.SpanIDFromHex(sp.spanID)

	spanContext := oteltrace.SpanContext{
		TraceID: traceID,
		SpanID:  spanID,
	}

	return otelSpan{
		spanContext: spanContext,
	}
}

// OCSpanContextToHCSpanContext converts an OpenTelemetry Go SDK core.SpanContext
// object into a Honeycomb SpanContext.
func OCSpanContextToHCSpanContext(spanContext oteltrace.SpanContext) *SpanContext {
	sc := &SpanContext{
		TraceID:    hex.EncodeToString(spanContext.TraceID[:]),
		ParentID:   hex.EncodeToString(spanContext.SpanID[:]),
		TraceState: spanContext.TraceFlags,
	}
	return sc
}
