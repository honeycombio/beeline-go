package trace

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTraceFromContext(t *testing.T) {
	ctx, tr := NewTrace(context.Background(), "")
	trInCtx := GetTraceFromContext(ctx)
	assert.Equal(t, tr, trInCtx, "trace from context should be the trace we got from making a new trace")
	emptyTrace := &Trace{}
	ctx = PutTraceInContext(ctx, emptyTrace)
	trInCtx = GetTraceFromContext(ctx)
	assert.Equal(t, emptyTrace, trInCtx, "trace in context should be trace we put in the context")
}

func TestSpanFromContext(t *testing.T) {
	ctx, tr := NewTrace(context.Background(), "")
	rs := tr.GetRootSpan()
	spanInCtx := GetSpanFromContext(ctx)
	assert.Equal(t, rs, spanInCtx, "span from context should be the root span we got from making a new trace")
	emptySpan := &Span{}
	ctx = PutSpanInContext(ctx, emptySpan)
	spanInCtx = GetSpanFromContext(ctx)
	assert.Equal(t, emptySpan, spanInCtx, "span in context should be span we put in the context")
}
