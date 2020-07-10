package propagation

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPropagationContextIsValid(t *testing.T) {
	// an empty propagation context is obviously invalid
	prop := &PropagationContext{}
	assert.False(t, prop.IsValid())

	// a propagation context with only a trace context is still invalid because it lacks a parent id and trace id
	prop = &PropagationContext{
		TraceContext: map[string]interface{}{
			"foo": "bar",
		},
	}
	assert.False(t, prop.IsValid())

	// a propagation context with a trace id but no parent id is invalid
	prop = &PropagationContext{
		TraceID: "trace_id",
	}
	assert.False(t, prop.IsValid())

	// as is the inverse (parent id but no trace id)
	prop = &PropagationContext{
		ParentID: "parent_id",
	}
	assert.False(t, prop.IsValid())

	// a propagation context is valid when it has a parent id and a trace id
	prop = &PropagationContext{
		ParentID: "parent_id",
		TraceID: "trace_id",
	}
	assert.True(t, prop.IsValid())

	// but not one that is the zero value for a byte array
	var spanID [8]byte
	var traceID [16]byte
	prop = &PropagationContext{
		ParentID: hex.EncodeToString(spanID[:]),
		TraceID: hex.EncodeToString(traceID[:]),
	}
	assert.Equal(t, false, prop.IsValid())
}

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
	assert.Error(t, err, "should not be able to unmarshal header without trace_id or parent_id")
}

func TestMarshalAmazonTraceContext(t *testing.T) {
	// According to the documentation for load balancer request tracing:
	// https://docs.aws.amazon.com/elasticloadbalancing/latest/application/load-balancer-request-tracing.html
	// An application can add arbitrary fields for its own purposes. The load balancer preserves these fields
	// but does not use them. In our implementation, we stick these fields in the TraceContext. Because of the
	// implementation, the TraceContext only supports strings whereas in the Honeycomb header format, these
	// fields are stored as base64 encoded JSON and therefore can support basic types like strings, booleans, etc.
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
	assert.Equal(t, "Root=abcdef123456;Self=0102030405", header[0:33])

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
		TraceID:  "invalid-trace-id",
		ParentID: "invalid-parent-id",
	}
	ctx, headers = MarshalW3CTraceContext(ctx, prop)
	assert.Equal(t, 0, len(headers))

	// ensure that roundtrip keeps tracestate intact
	headers = map[string]string{
		"traceparent": "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-00",
		"tracestate":  "foo=bar,bar=baz",
	}
	ctx, prop, err := UnmarshalW3CTraceContext(ctx, headers)
	assert.NoError(t, err, "unmarshal w3c headers")
	ctx, marshaled := MarshalW3CTraceContext(ctx, prop)
	assert.Equal(t, "foo=bar,bar=baz", marshaled["tracestate"])

	// ensure that empty headers are handled the way we expect (silently)
	headers = map[string]string{}
	ctx, prop, err = UnmarshalW3CTraceContext(context.Background(), headers)
	assert.Error(t, err, "Cannot unmarshal empty header")
}

func TestUnmarshalTraceContext(t *testing.T) {
	testCases := []struct {
		name       string
		contextStr string
		prop       *PropagationContext
		returnsErr bool
	}{
		{
			"empty header- we expect an error because the version string will be invalid",
			"",
			nil,
			true,
		},
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
			"v1, missing parent_id, should return an error",
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
			"empty header - throw an error since it contains neither a trace id nor parent id",
			"",
			nil,
			true,
		},
		{
			"all fields legit",
			"Root=1-67891233-abcdef012345678912345678;Self=1-67891233-abcdef0876543219876543210",
			&PropagationContext{
				TraceID:      "1-67891233-abcdef012345678912345678",
				ParentID:     "1-67891233-abcdef0876543219876543210",
				TraceContext: make(map[string]interface{}),
			},
			false,
		},
		{
			"all fields legit with some context",
			"Root=1-67891233-abcdef012345678912345678;Self=1-67891233-abcdef0876543219876543210;Foo=bar;UserId=123;toRetry=true",
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
			"self, parent and root fields. parent should end up in trace context",
			"Root=foo;Parent=bar;Self=baz",
			&PropagationContext{
				TraceID:      "foo",
				ParentID:     "baz",
				TraceContext: map[string]interface{}{
					"Parent": "bar",
				},
			},
			false,
		},
		{
			"self, parent and root fields. parent should end up in trace context",
			"Root=foo;Self=baz;Parent=bar",
			&PropagationContext{
				TraceID:      "foo",
				ParentID:     "baz",
				TraceContext: map[string]interface{}{
					"Parent": "bar",
				},
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
			"Foo=bar;Self=foobar;Bar=baz",
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
