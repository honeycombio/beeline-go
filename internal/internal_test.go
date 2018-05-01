package internal

import (
	"net/http/httptest"
	"testing"

	libhoney "github.com/honeycombio/libhoney-go"
	"github.com/stretchr/testify/assert"
)

func TestParseTraceHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Amzn-Trace-Id", "Self=1-67891234-12456789abcdef012345678;Root=1-67891233-abcdef012345678912345678;CalledFrom=app")
	ev := libhoney.NewEvent()
	parseTraceHeader(req, ev)
	fs := ev.Fields()
	assert.Equal(t, fs["request.trace_id.Self"], "1-67891234-12456789abcdef012345678")
	assert.Equal(t, fs["request.trace_id.Root"], "1-67891233-abcdef012345678912345678")
	assert.Equal(t, fs["request.trace_id.CalledFrom"], "app")
}
