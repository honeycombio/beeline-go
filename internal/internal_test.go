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
	// spew.Dump(fs)
	assert.Equal(t, "1-67891234-12456789abcdef012345678", fs["request.header.aws_trace_id.Self"])
	assert.Equal(t, "1-67891233-abcdef012345678912345678", fs["request.header.aws_trace_id.Root"])
	assert.Equal(t, "app", fs["request.header.aws_trace_id.CalledFrom"])
}

func TestHostHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Host", "example.com")
	ev := libhoney.NewEvent()
	AddRequestProps(req, ev)
	fs := ev.Fields()
	assert.Equal(t, "example.com", fs["request.host"])
}

func TestURLHostHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "https://example.com/", nil)
	ev := libhoney.NewEvent()
	AddRequestProps(req, ev)
	fs := ev.Fields()
	assert.Equal(t, "example.com", fs["request.host"])
}

func TestXForwardedForHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "https://example.com/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	ev := libhoney.NewEvent()
	AddRequestProps(req, ev)
	fs := ev.Fields()
	assert.Equal(t, "1.2.3.4", fs["request.header.x_forwarded_for"])
}

func TestXForwardedProtoHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "https://example.com/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	ev := libhoney.NewEvent()
	AddRequestProps(req, ev)
	fs := ev.Fields()
	assert.Equal(t, "https", fs["request.header.x_forwarded_proto"])
}
