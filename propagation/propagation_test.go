package propagation

import (
	"context"
	"testing"

	"github.com/honeycombio/beeline-go/trace"
	"github.com/stretchr/testify/assert"
)

// MockRequest is used as an HTTPSupplier when parsing and injecting headers.
type MockRequest struct {
	values map[string]string
}

func NewMockRequest() *MockRequest {
	m := &MockRequest{}
	m.values = make(map[string]string)
	return m
}

func (m MockRequest) Get(key string) string {
	if value, ok := m.values[key]; ok {
		return value
	}
	return ""
}

func (m MockRequest) Set(key string, value string) {
	m.values[key] = value
}

func TestMarshalHoneycombTraceContext(t *testing.T) {
	sc := &trace.SpanContext{
		TraceID:  "abcdef123456",
		ParentID: "0102030405",
		TraceContext: map[string]interface{}{
			"userID":   float64(1),
			"errorMsg": "failed to sign on",
			"toRetry":  true,
		},
	}

	m := NewMockRequest()
	ctx := trace.PutRemoteSpanContextInContext(context.Background(), sc)
	propagator := HoneycombHTTPPropagator{}
	propagator.Insert(ctx, m)
	marshaled := m.Get(honeycombTracePropagationHTTPHeader)
	assert.Equal(t, "1;", marshaled[0:2], "version of marshaled context should be 1")
	assert.Equal(t, "1;trace_id=abcdef123456,parent_id=0102030405,context=eyJlcnJvck1zZyI6ImZhaWxlZCB0byBzaWduIG9uIiwidG9SZXRyeSI6dHJ1ZSwidXNlcklEIjoxfQ==", marshaled)

	returned := propagator.Parse(ctx, m)

	assert.Equal(t, sc, returned, "roundtrip object")

	sc.Dataset = "imadataset"
	propagator.Insert(ctx, m)
	marshaled = m.Get(honeycombTracePropagationHTTPHeader)
	assert.Equal(t, "1;", marshaled[0:2], "version of marshaled context should be 1")
	assert.Equal(t, "1;trace_id=abcdef123456,parent_id=0102030405,dataset=imadataset,context=eyJlcnJvck1zZyI6ImZhaWxlZCB0byBzaWduIG9uIiwidG9SZXRyeSI6dHJ1ZSwidXNlcklEIjoxfQ==", marshaled)

	returned = propagator.Parse(ctx, m)
	assert.Equal(t, sc, returned, "roundtrip object")

	sc.Dataset = "ill;egal"
	propagator.Insert(ctx, m)
	marshaled = m.Get(honeycombTracePropagationHTTPHeader)
	assert.Equal(t, "1;", marshaled[0:2], "version of marshaled context should be 1")
	assert.Equal(t, "1;trace_id=abcdef123456,parent_id=0102030405,dataset=ill%3Begal,context=eyJlcnJvck1zZyI6ImZhaWxlZCB0byBzaWduIG9uIiwidG9SZXRyeSI6dHJ1ZSwidXNlcklEIjoxfQ==", marshaled)

	returned = propagator.Parse(ctx, m)
	assert.Equal(t, sc, returned, "roundtrip object")
	sc = &trace.SpanContext{
		Dataset: "imadataset",
	}
	ctx = trace.PutRemoteSpanContextInContext(ctx, sc)
	propagator.Insert(ctx, m)
	marshaled = m.Get(honeycombTracePropagationHTTPHeader)
	assert.Equal(t, "1;", marshaled[0:2], "version of marshaled context should be 1")
	assert.Equal(t, "1;trace_id=,parent_id=,dataset=imadataset,context=bnVsbA==", marshaled)

	returned = propagator.Parse(ctx, m)
	assert.Equal(t, sc, returned, "roundtrip object")
}

func TestMarshalAmazonTraceContext(t *testing.T) {
	// NOTE: we only support strings for trace context in amazon headers
	sc := &trace.SpanContext{
		TraceID:  "abcdef123456",
		ParentID: "0102030405",
		TraceContext: map[string]interface{}{
			"userID":   "1",
			"errorMsg": "failed to sign on",
			"toRetry":  "true",
		},
	}

	m := NewMockRequest()
	ctx := trace.PutRemoteSpanContextInContext(context.Background(), sc)
	propagator := AmazonHTTPPropagator{}
	propagator.Insert(ctx, m)
	marshaled := m.Get(amazonTracePropagationHTTPHeader)
	assert.Equal(t, "Root=abcdef123456;Parent=0102030405", marshaled[0:35])

	returned := propagator.Parse(ctx, m)
	assert.Equal(t, sc, returned, "roundtrip object")
}

func TestUnmarshalHoneycombTraceContext(t *testing.T) {
	testCases := []struct {
		name       string
		contextStr string
		sc         *trace.SpanContext
		returnsErr bool
	}{
		{
			"unsupported version",
			"999999;....",
			nil,
			true,
		},
		{
			"v1 trace_id + parent_id, missing context",
			"1;trace_id=abcdef,parent_id=12345",
			&trace.SpanContext{
				TraceID:  "abcdef",
				ParentID: "12345",
			},
			false,
		},
		{
			"v1, all headers and legit context",
			"1;trace_id=abcdef,parent_id=12345,context=eyJlcnJvck1zZyI6ImZhaWxlZCB0byBzaWduIG9uIiwidG9SZXRyeSI6dHJ1ZSwidXNlcklEIjoxfQ==",
			&trace.SpanContext{
				TraceID:  "abcdef",
				ParentID: "12345",
				TraceContext: map[string]interface{}{
					"userID":   float64(1),
					"errorMsg": "failed to sign on",
					"toRetry":  true,
				},
			},
			false,
		},
		{
			"v1, parent_id without trace_id",
			"1;parent_id=12345",
			nil,
			true,
		},
		{
			"v1, missing parent_id",
			"1;trace_id=12345",
			&trace.SpanContext{
				TraceID: "12345",
			},
			false,
		},
		{
			"v1, garbled context",
			"1;trace_id=abcdef,parent_id=12345,context=123~!@@&^@",
			nil,
			true,
		},
		{
			"v1, unknown key (otherwise valid)",
			"1;trace_id=abcdef,parent_id=12345,something=unsupported",
			&trace.SpanContext{
				TraceID:  "abcdef",
				ParentID: "12345",
			},
			false,
		},
		{
			"v1, extra unknown key (otherwise valid)",
			"1;trace_id=abcdef,parent_id=12345,something=unsupported,context=eyJlcnJvck1zZyI6ImZhaWxlZCB0byBzaWduIG9uIiwidG9SZXRyeSI6dHJ1ZSwidXNlcklEIjoxfQ==",
			&trace.SpanContext{
				TraceID:  "abcdef",
				ParentID: "12345",
				TraceContext: map[string]interface{}{
					"userID":   float64(1),
					"errorMsg": "failed to sign on",
					"toRetry":  true,
				},
			},
			false,
		},
	}

	m := NewMockRequest()
	propagator := HoneycombHTTPPropagator{}
	ctx := context.Background()
	for _, tt := range testCases {
		m.Set("X-Honeycomb-Trace", tt.contextStr)
		sc := propagator.Parse(ctx, m)
		assert.Equal(t, tt.sc, sc, tt.name)
	}
}

func TestUnmarshalAmazonTraceContext(t *testing.T) {
	testCases := []struct {
		name       string
		contextStr string
		sc         *trace.SpanContext
		returnsErr bool
	}{
		{
			"root present, no self or parent",
			"Root=foobar",
			&trace.SpanContext{
				TraceID:      "foobar",
				ParentID:     "foobar",
				TraceContext: make(map[string]interface{}),
			},
			false,
		},
		{
			"root, self present",
			"Root=foobar;Self=barbaz",
			&trace.SpanContext{
				TraceID:      "foobar",
				ParentID:     "barbaz",
				TraceContext: make(map[string]interface{}),
			},
			false,
		},
		{
			"root, self and parent present",
			"Root=foobar;Self=barbaz;Parent=foobaz",
			&trace.SpanContext{
				TraceID:      "foobar",
				ParentID:     "foobaz",
				TraceContext: make(map[string]interface{}),
			},
			false,
		},
		{
			"Missing parent and trace id",
			"Self=foobar",
			nil,
			true,
		},
		{
			"Extra fields in trace context",
			"Root=foobarbaz;Foo=Bar;Something=1",
			&trace.SpanContext{
				TraceID:  "foobarbaz",
				ParentID: "foobarbaz",
				TraceContext: map[string]interface{}{
					"Foo":       "Bar",
					"Something": "1",
				},
			},
			false,
		},
	}

	m := NewMockRequest()
	propagator := AmazonHTTPPropagator{}
	ctx := context.Background()
	for _, tt := range testCases {
		m.Set("X-Amzn-Trace-Id", tt.contextStr)
		sc := propagator.Parse(ctx, m)
		assert.Equal(t, tt.sc, sc, tt.name)
	}
}
