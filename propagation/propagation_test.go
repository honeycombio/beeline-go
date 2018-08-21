package propagation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// var testHeaders = TraceHeader{
// 	Source:   HeaderSourceBeeline,
// 	TraceID:  "abcdef123456",
// 	ParentID: "0102030405",
// }
// var testTrace = &Trace{
// 	headers: testHeaders,
// 	spans:   []*Span{},

// 	traceLevelFields: map[string]interface{}{
// "userID":   float64(1),
// "errorMsg": "failed to sign on",
// "toRetry":  true,
// 	},
// }
// var testSpan = &Span{spanID: "0102030405"}

// func init() {
// 	// set up the links correctly
// 	testTrace.AddSpan(testSpan)
// 	testSpan.trace = testTrace
// }

func TestMarshalTraceContext(t *testing.T) {
	prop := &Propagation{
		TraceID:  "abcdef123456",
		ParentID: "0102030405",
		TraceContext: map[string]interface{}{
			"userID":   float64(1),
			"errorMsg": "failed to sign on",
			"toRetry":  true,
		},
	}

	marshaled := MarshalTraceContext(prop)
	assert.Equal(t, "1;", marshaled[0:2], "version of marshaled context should be 1")
	assert.Equal(t, "1;trace_id=abcdef123456,parent_id=0102030405,context=eyJlcnJvck1zZyI6ImZhaWxlZCB0byBzaWduIG9uIiwidG9SZXRyeSI6dHJ1ZSwidXNlcklEIjoxfQ==", marshaled)
}

func TestUnmarshalTraceContext(t *testing.T) {
	testCases := []struct {
		name       string
		contextStr string
		prop       *Propagation
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
			&Propagation{
				TraceID:  "abcdef",
				ParentID: "12345",
			},
			false,
		},
		{
			"v1, all headers and legit context",
			"1;trace_id=abcdef,parent_id=12345,context=eyJlcnJvck1zZyI6ImZhaWxlZCB0byBzaWduIG9uIiwidG9SZXRyeSI6dHJ1ZSwidXNlcklEIjoxfQ==",
			&Propagation{
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
			"v1, missing trace_id",
			"1;parent_id=12345",
			nil,
			true,
		},
		{
			"v1, missing parent_id",
			"1;trace_id=12345",
			nil,
			true,
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
			&Propagation{
				TraceID:  "abcdef",
				ParentID: "12345",
			},
			false,
		},
		{
			"v1, extra unknown key (otherwise valid)",
			"1;trace_id=abcdef,parent_id=12345,something=unsupported,context=eyJlcnJvck1zZyI6ImZhaWxlZCB0byBzaWduIG9uIiwidG9SZXRyeSI6dHJ1ZSwidXNlcklEIjoxfQ==",
			&Propagation{
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

	for _, tt := range testCases {
		prop, err := UnmarshalTraceContext(tt.contextStr)
		assert.Equal(t, tt.prop, prop, tt.name)
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
	prop := &Propagation{
		TraceID:  "abcdef123456",
		ParentID: "0102030405",
		TraceContext: map[string]interface{}{
			"userID":   float64(1),
			"errorMsg": "failed to sign on",
			"toRetry":  true,
		},
	}
	marshaled := MarshalTraceContext(prop)
	returned, err := UnmarshalTraceContext(marshaled)
	assert.Equal(t, prop, returned, "roundtrip object")
	assert.NoError(t, err, "roundtrip error")
}
