package propagation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarshalTraceContext(t *testing.T) {
	prop := &PropagationContext{
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

	returned, err := UnmarshalTraceContext(marshaled)
	assert.Equal(t, prop, returned, "roundtrip object")
	assert.NoError(t, err, "roundtrip error")

	prop.Dataset = "imadataset"
	marshaled = MarshalTraceContext(prop)
	assert.Equal(t, "1;", marshaled[0:2], "version of marshaled context should be 1")
	assert.Equal(t, "1;trace_id=abcdef123456,parent_id=0102030405,dataset=imadataset,context=eyJlcnJvck1zZyI6ImZhaWxlZCB0byBzaWduIG9uIiwidG9SZXRyeSI6dHJ1ZSwidXNlcklEIjoxfQ==", marshaled)

	returned, err = UnmarshalTraceContext(marshaled)
	assert.Equal(t, prop, returned, "roundtrip object")
	assert.NoError(t, err, "roundtrip error")

	prop.Dataset = "ill;egal"
	marshaled = MarshalTraceContext(prop)
	assert.Equal(t, "1;", marshaled[0:2], "version of marshaled context should be 1")
	assert.Equal(t, "1;trace_id=abcdef123456,parent_id=0102030405,dataset=ill%3Begal,context=eyJlcnJvck1zZyI6ImZhaWxlZCB0byBzaWduIG9uIiwidG9SZXRyeSI6dHJ1ZSwidXNlcklEIjoxfQ==", marshaled)

	returned, err = UnmarshalTraceContext(marshaled)
	assert.Equal(t, prop, returned, "roundtrip object")
	assert.NoError(t, err, "roundtrip error")

	prop = &PropagationContext{
		Dataset: "imadataset",
	}
	marshaled = MarshalTraceContext(prop)
	assert.Equal(t, "1;", marshaled[0:2], "version of marshaled context should be 1")
	assert.Equal(t, "1;trace_id=,parent_id=,dataset=imadataset,context=bnVsbA==", marshaled)

	returned, err = UnmarshalTraceContext(marshaled)
	assert.Equal(t, prop, returned, "roundtrip object")
	assert.NoError(t, err, "roundtrip error")
}

func TestMarshalAmazonTraceContext(t *testing.T) {
	// NOTE: we only support strings for trace context in amazon headers
	prop := &PropagationContext{
		TraceID:  "abcdef123456",
		ParentID: "0102030405",
		TraceContext: map[string]interface{}{
			"userID":   "1",
			"errorMsg": "failed to sign on",
			"toRetry":  "true",
		},
	}

	header := MarshalAmazonTraceContext(prop)
	// Note: we don't test trace context because we can't gaurantee the order.
	// It's covered by the roundtrip test below.
	assert.Equal(t, "Root=abcdef123456;Parent=0102030405", header[0:35])

	returned, err := UnmarshalAmazonTraceContext(header)
	if assert.NoError(t, err) {
		assert.Equal(t, prop, returned, "roundtrip object")
	}
}

func TestW3CTraceContext(t *testing.T) {
	prop := &PropagationContext{
		TraceID:  "0af7651916cd43dd8448eb211c80319c",
		ParentID: "b7ad6b7169203331",
	}
	ctx, headers := MarshalW3CTraceContext(context.Background(), prop)
	assert.Equal(t, 2, len(headers), "W3C Trace Context should have two headers")
	assert.Equal(t, "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-00", headers["traceparent"])
	// should result in empty headers
	prop = &PropagationContext{
		TraceID: "invalid-trace-id",
		ParentID: "invalid-parent-id",
	}
	ctx, headers = MarshalW3CTraceContext(ctx, prop)
	assert.Equal(t, 0, len(headers))

	// ensure that roundtrip keeps tracestate intact
	headers = map[string]string{
		"traceparent": "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-00",
		"tracestate": "foo=bar,bar=baz",
	}
	ctx, prop = UnmarshalW3CTraceContext(ctx, headers)
	ctx, marshaled := MarshalW3CTraceContext(ctx, prop)
	assert.Equal(t, "foo=bar,bar=baz", marshaled["tracestate"])
}

func TestUnmarshalTraceContext(t *testing.T) {
	testCases := []struct {
		name       string
		contextStr string
		prop       *PropagationContext
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
			&PropagationContext{
				TraceID:  "abcdef",
				ParentID: "12345",
			},
			false,
		},
		{
			"v1, all headers and legit context",
			"1;trace_id=abcdef,parent_id=12345,context=eyJlcnJvck1zZyI6ImZhaWxlZCB0byBzaWduIG9uIiwidG9SZXRyeSI6dHJ1ZSwidXNlcklEIjoxfQ==",
			&PropagationContext{
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
			&PropagationContext{
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
			&PropagationContext{
				TraceID:  "abcdef",
				ParentID: "12345",
			},
			false,
		},
		{
			"v1, extra unknown key (otherwise valid)",
			"1;trace_id=abcdef,parent_id=12345,something=unsupported,context=eyJlcnJvck1zZyI6ImZhaWxlZCB0byBzaWduIG9uIiwidG9SZXRyeSI6dHJ1ZSwidXNlcklEIjoxfQ==",
			&PropagationContext{
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

func TestUnmarshalAmazonTraceContext(t *testing.T) {
	testCases := []struct {
		name       string
		contextStr string
		prop       *PropagationContext
		returnsErr bool
	}{
		{
			"all fields legit",
			"Root=1-67891233-abcdef012345678912345678;Parent=1-67891233-abcdef0876543219876543210",
			&PropagationContext{
				TraceID:      "1-67891233-abcdef012345678912345678",
				ParentID:     "1-67891233-abcdef0876543219876543210",
				TraceContext: make(map[string]interface{}),
			},
			false,
		},
		{
			"all fields legit with some context",
			"Root=1-67891233-abcdef012345678912345678;Parent=1-67891233-abcdef0876543219876543210;Foo=bar;UserId=123;toRetry=true",
			&PropagationContext{
				TraceID:  "1-67891233-abcdef012345678912345678",
				ParentID: "1-67891233-abcdef0876543219876543210",
				TraceContext: map[string]interface{}{
					"Foo":     "bar",
					"UserId":  "123",
					"toRetry": "true",
				},
			},
			false,
		},
		{
			"self, parent and root fields. last should win",
			"Root=foo;Parent=bar;Self=baz",
			&PropagationContext{
				TraceID:      "foo",
				ParentID:     "baz",
				TraceContext: make(map[string]interface{}),
			},
			false,
		},
		{
			"self, parent and root fields. last should win",
			"Root=foo;Self=baz;Parent=bar",
			&PropagationContext{
				TraceID:      "foo",
				ParentID:     "bar",
				TraceContext: make(map[string]interface{}),
			},
			false,
		},
		{
			"Missing trace id, should inherit parent id",
			"Root=foo;Foo=bar",
			&PropagationContext{
				TraceID:  "foo",
				ParentID: "foo",
				TraceContext: map[string]interface{}{
					"Foo": "bar",
				},
			},
			false,
		},
		{
			"Missing trace id and parent id is populated, error",
			"Foo=bar;Parent=foobar;Bar=baz",
			nil,
			true,
		},
	}

	for _, tt := range testCases {
		prop, err := UnmarshalAmazonTraceContext(tt.contextStr)
		assert.Equal(t, tt.prop, prop, tt.name)
		if tt.returnsErr {
			assert.Error(t, err, tt.name)
		} else {
			assert.NoError(t, err, tt.name)
		}
	}
}
