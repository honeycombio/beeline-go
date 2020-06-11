package propagation

import (
	"context"

	"github.com/honeycombio/beeline-go/trace"
	oteltrace "go.opentelemetry.io/otel/api/trace"
)

// W3CHTTPPropagator is used to extract and generate / insert W3C Trace Context headers.
type W3CHTTPPropagator struct{}

// Parse uses an OpenTelemetry Default HTTP Propagator to extract the W3C Trace Context headers from
// header and creates a SpanContext object. The SpanContext is stored in ctx and returned by this method.
func (wcp W3CHTTPPropagator) Parse(ctx context.Context, header trace.HeaderSupplier) *trace.SpanContext {
	prop := oteltrace.DefaultHTTPPropagator()
	ctx = prop.Extract(ctx, header)
	return trace.GetRemoteSpanContextFromContext(ctx)
}

// Insert uses an OpenTelemetry Default HTTP Propagator to extract an OpenTelemetry Span object from ctx
// and use th einformation contained within to serialize the appropriate W3C Trace Context headers,
// injecting them into header.
func (wcp W3CHTTPPropagator) Insert(ctx context.Context, header trace.HeaderSupplier) {
	// HTTP Propagators that are defined in the OpenTelemetry Go SDK expect
	// a context that has an OpenTelemetry Span object. To be able to use
	// those propagators here, we create an OpenTelemetry Span object and
	// store it in the context.
	otelSpan := trace.OTelSpanFromContext(ctx)
	ctx = oteltrace.ContextWithSpan(ctx, otelSpan)
	prop := oteltrace.DefaultHTTPPropagator()
	prop.Inject(ctx, header)
}
