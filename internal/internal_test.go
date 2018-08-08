package internal

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTraceHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Amzn-Trace-Id", "Self=1-67891234-12456789abcdef012345678;Root=1-67891233-abcdef012345678912345678;CalledFrom=app")
	headers, _, err := FindTraceHeaders(req)
	assert.NoError(t, err)
	// spew.Dump(fs)
	assert.Equal(t, HeaderSourceAmazon, headers.Source, "didn't identify amazon as the source of headers")
	// assert.Equal(t, "1-67891234-12456789abcdef012345678", fs["request.header.aws_trace_id.Self"])
	assert.Equal(t, "1-67891233-abcdef012345678912345678", headers.TraceID)
	// assert.Equal(t, "app", fs["request.header.aws_trace_id.CalledFrom"])
}
