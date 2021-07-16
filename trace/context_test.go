package trace

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTraceFromContext(t *testing.T) {
	ctx, tr := NewTrace(context.Background(), nil)
	trInCtx := GetTraceFromContext(ctx)
	assert.Equal(t, tr, trInCtx, "trace from context should be the trace we got from making a new trace")
	emptyTrace := &Trace{}
	ctx = PutTraceInContext(ctx, emptyTrace)
	trInCtx = GetTraceFromContext(ctx)
	assert.Equal(t, emptyTrace, trInCtx, "trace in context should be trace we put in the context")
}

func TestSpanFromContext(t *testing.T) {
	ctx, tr := NewTrace(context.Background(), nil)
	rs := tr.GetRootSpan()
	spanInCtx := GetSpanFromContext(ctx)
	assert.Equal(t, rs, spanInCtx, "span from context should be the root span we got from making a new trace")
	emptySpan := &Span{}
	ctx = PutSpanInContext(ctx, emptySpan)
	spanInCtx = GetSpanFromContext(ctx)
	assert.Equal(t, emptySpan, spanInCtx, "span in context should be span we put in the context")
}

func TestCopyContext(t *testing.T) {
	ctx, tr := NewTrace(context.Background(), nil)
	rs := tr.GetRootSpan()

	newCtx, err := CopyContext(context.Background(), ctx)
	assert.NoError(t, err, "should not return error when trace and span are present")

	trInCtx := GetTraceFromContext(newCtx)
	spanInCtx := GetSpanFromContext(newCtx)

	assert.Equal(t, trInCtx, tr, "expected to find the same trace in the new context after copy")
	assert.Equal(t, spanInCtx, rs, "expected to find the same span in the new context after copy")
}

func TestCopyContextError(t *testing.T) {
	newCtx, err := CopyContext(context.Background(), context.Background())
	assert.NotNil(t, newCtx, "should return valid context even in errored state")
	assert.Equal(t, err, ErrTraceNotFoundInContext, "should error when no trace is present in the context")

}
