package internal

import "context"

const (
	honeyBuilderContextKey = "honeycombBuilderContextKey"
	honeyEventContextKey   = "honeycombEventContextKey"
	honeyTraceContextKey   = "honeycombTraceContextKey"
)

// GetTraceFromContext pulls a trace off the passed in context or returns nil if
// no trace exists.
func GetTraceFromContext(ctx context.Context) *Trace {
	if ctx != nil {
		if trace, ok := ctx.Value(honeyTraceContextKey).(*Trace); ok {
			return trace
		}
	}
	return nil
}

// PutTraceInContext takes an existing context and a trace and pushes the trace
// into the context.  It should replace any traces that already exist in the
// context. The returned error will be not nil if a trace already existed.
func PutTraceInContext(ctx context.Context, trace *Trace) (context.Context, error) {
	return context.WithValue(ctx, honeyTraceContextKey, trace), nil
}
