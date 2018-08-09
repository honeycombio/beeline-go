package internal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testHeaders = TraceHeader{
	Source:   HeaderSourceBeeline,
	TraceID:  "abcdef123456",
	ParentID: "0102030405",
}
var testTrace = &Trace{
	headers: testHeaders,
	openSpans: []*Span{
		&Span{
			spanID: "0102030405",
		},
	},
	traceLevelFields: map[string]interface{}{
		"userID":   float64(1),
		"errorMsg": "failed to sign on",
		"toRetry":  true,
	},
}

func TestMarshalTraceContext(t *testing.T) {
	ctx, err := PutTraceInContext(context.TODO(), testTrace)
	assert.Nil(t, err, "Put trace in context should not error")
	marshaled := MarshalTraceContext(ctx)
	assert.Equal(t, "1;", marshaled[0:2])
}

func TestUnmarshalTraceContext(t *testing.T) {
	testCases := []struct {
		name         string
		contextStr   string
		valueHeader  *TraceHeader
		valueContext map[string]interface{}
		returnsErr   bool
	}{
		{
			"unsupported version",
			"999999;....",
			nil,
			nil,
			true,
		},
		{
			"v1 trace_id + parent_id, missing context",
			"1;trace_id=abcdef,parent_id=12345",
			&TraceHeader{
				Source:   HeaderSourceBeeline,
				TraceID:  "abcdef",
				ParentID: "12345",
			},
			nil,
			false,
		},
		{
			"v1, all headers and legit context",
			"1;trace_id=abcdef,parent_id=12345,context=eyJlcnJvck1zZyI6ImZhaWxlZCB0byBzaWduIG9uIiwidG9SZXRyeSI6dHJ1ZSwidXNlcklEIjoxfQ==",
			&TraceHeader{
				Source:   HeaderSourceBeeline,
				TraceID:  "abcdef",
				ParentID: "12345",
			},
			map[string]interface{}{
				"userID":   float64(1),
				"errorMsg": "failed to sign on",
				"toRetry":  true,
			},
			false,
		},
		{
			"v1, missing trace_id",
			"1;parent_id=12345",
			nil,
			nil,
			true,
		},
		{
			"v1, missing parent_id",
			"1;trace_id=12345",
			nil,
			nil,
			true,
		},
		{
			"v1, garbled context",
			"1;trace_id=abcdef,parent_id=12345,context=123~!@@&^@",
			nil,
			nil,
			true,
		},
		{
			"v1, unknown key (otherwise valid)",
			"1;trace_id=abcdef,parent_id=12345,something=unsupported",
			&TraceHeader{
				Source:   HeaderSourceBeeline,
				TraceID:  "abcdef",
				ParentID: "12345",
			},
			nil,
			false,
		},
		{
			"v1, extra unknown key (otherwise valid)",
			"1;trace_id=abcdef,parent_id=12345,something=unsupported,context=eyJlcnJvck1zZyI6ImZhaWxlZCB0byBzaWduIG9uIiwidG9SZXRyeSI6dHJ1ZSwidXNlcklEIjoxfQ==",
			&TraceHeader{
				Source:   HeaderSourceBeeline,
				TraceID:  "abcdef",
				ParentID: "12345",
			},
			map[string]interface{}{
				"userID":   float64(1),
				"errorMsg": "failed to sign on",
				"toRetry":  true,
			},
			false,
		},
	}

	for _, tt := range testCases {
		header, fields, err := UnmarshalTraceContext(tt.contextStr)
		assert.Equal(t, tt.valueHeader, header, tt.name)
		assert.Equal(t, tt.valueContext, fields, tt.name)
		if tt.returnsErr {
			assert.Error(t, err, tt.name)
		} else {
			assert.NoError(t, err, tt.name)
		}
	}
}

// TestContextPropagationRoundTrip encodes some things then decodes them and
// expects to get back the same thing it put in
func TestContextPropagationRoundTrip(t *testing.T) {
	ctx, err := PutTraceInContext(context.TODO(), testTrace)
	assert.Nil(t, err, "Put trace in context should not error")
	marshaled := MarshalTraceContext(ctx)
	header, fields, err := UnmarshalTraceContext(marshaled)
	assert.Equal(t, &testHeaders, header, "roundtrip headers")
	assert.Equal(t, testTrace.traceLevelFields, fields, "roundtrip context")
	assert.NoError(t, err, "roundtrip error")
}
