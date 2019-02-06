package trace

import (
	"context"
	"errors"

	"github.com/honeycombio/beeline-go/trace"
)

const (
	honeySpanContextKey  = "honeycombSpanContextKey"
	honeyTraceContextKey = "honeycombTraceContextKey"
)

var (
	ErrTraceNotFoundInContext = errors.New("beeline trace not found in source context")
)

// GetTraceFromContext retrieves a trace from the passed in context or returns
// nil if no trace exists.
func GetTraceFromContext(ctx context.Context) *Trace {
	if ctx != nil {
		if val := ctx.Value(honeyTraceContextKey); val != nil {
			if trace, ok := val.(*Trace); ok {
				return trace
			}
		}
	}
	return nil
}

// PutTraceInContext takes an existing context and a trace and pushes the trace
// into the context.  It will replace any traces that already exist in the
// context. Traces put in context are retrieved using GetTraceFromContext.
func PutTraceInContext(ctx context.Context, trace *Trace) context.Context {
	return context.WithValue(ctx, honeyTraceContextKey, trace)
}

// GetSpanFromContext identifies the currently active span via the span context
// key. It returns that span, and access to the trace is available via the span
// or from the context directly. It will return nil if there is no span
// available.
func GetSpanFromContext(ctx context.Context) *Span {
	if ctx != nil {
		if val := ctx.Value(honeySpanContextKey); val != nil {
			if span, ok := val.(*Span); ok {
				return span
			}
		}
	}
	return nil
}

// PutSpanInContext takes an existing context and a span and pushes the span
// into the context.  It will replace any spans that already exist in the
// context. Spans put in context are retrieved using GetSpanFromContext.
func PutSpanInContext(ctx context.Context, span *Span) context.Context {
	return context.WithValue(ctx, honeySpanContextKey, span)
}

// CopyContext takes a context that has a beeline trace and one that
// doesn't. It copies all the bits necessary to continue the trace from one to
// the other. This is useful if you need to break context to launch a goroutine
// that shouldn't be cancelled by the parent's cancellation context. It returns
// the newly populated context and an error if it can't find a trace in the
// source context.
func CopyContext(dest context.Context, src context.Context) context.Context, error {
	trace := trace.GetTraceFromContext(src)
	span := trace.GetSpanFromContext(src)
	if trace == nil || span == nil {
		return ErrTraceNotFoundInContext
	}
	dest = trace.PutTraceInContext(dest, trace)
	dest = trace.PutSpanInContext(dest, span)
}
