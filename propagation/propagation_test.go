package propagation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

	prop = &Propagation{
		Dataset: "imadataset",
	}
	marshaled = MarshalTraceContext(prop)
	assert.Equal(t, "1;", marshaled[0:2], "version of marshaled context should be 1")
	assert.Equal(t, "1;trace_id=,parent_id=,dataset=imadataset,context=bnVsbA==", marshaled)

	returned, err = UnmarshalTraceContext(marshaled)
	assert.Equal(t, prop, returned, "roundtrip object")
	assert.NoError(t, err, "roundtrip error")
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
			"v1, parent_id without trace_id",
			"1;parent_id=12345",
			nil,
			true,
		},
		{
			"v1, missing parent_id",
			"1;trace_id=12345",
			&Propagation{
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

func TestUnmarshalAWSTraceContext(t *testing.T) {
	testCases := []struct {
		name       string
		header     string
		prop       *Propagation
		returnsErr bool
	}{
		{
			"root / no parent",
			"Root=1-67891233-abcdef012345678912345678",
			&Propagation{
				TraceID:  "1-67891233-abcdef012345678912345678",
				ParentID: "1-67891233-abcdef012345678912345678",
			},
			false,
		},
		{
			"root / parent",
			"Root=1-5759e988-bd862e3fe1be46a994272793;Parent=53995c3f42cd8ad8",
			&Propagation{
				TraceID:  "1-5759e988-bd862e3fe1be46a994272793",
				ParentID: "53995c3f42cd8ad8",
			},
			false,
		},
		{
			"self / root / no parent",
			"Self=1-5983f5c9-36d365bc453d28036a63032b;Root=1-5983f5c9-56dcf0bc6d4d214d2dbbe8c6",
			&Propagation{
				TraceID:  "1-5983f5c9-56dcf0bc6d4d214d2dbbe8c6",
				ParentID: "1-5983f5c9-56dcf0bc6d4d214d2dbbe8c6",
			},
			false,
		},
		{
			"no root / parent",
			"Parent=53995c3f42cd8ad8",
			nil,
			true,
		},
	}

	for _, tt := range testCases {
		prop, err := UnmarshalAWSTraceContext(tt.header)
		assert.Equal(t, tt.prop, prop, tt.name)
		if tt.returnsErr {
			assert.Error(t, err, tt.name)
		} else {
			assert.NoError(t, err, tt.name)
		}
	}
}
