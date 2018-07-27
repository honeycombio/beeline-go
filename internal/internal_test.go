package internal

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTraceHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Amzn-Trace-Id", "Self=1-67891234-12456789abcdef012345678;Root=1-67891233-abcdef012345678912345678;CalledFrom=app")
	headers := FindTraceHeaders(req)
	// spew.Dump(fs)
	assert.Equal(t, HeaderSourceAmazon, headers.Source, "didn't identify amazon as the source of headers")
	// assert.Equal(t, "1-67891234-12456789abcdef012345678", fs["request.header.aws_trace_id.Self"])
	assert.Equal(t, "1-67891233-abcdef012345678912345678", headers.TraceID)
	// assert.Equal(t, "app", fs["request.header.aws_trace_id.CalledFrom"])
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

func TestUserAgentHeader(t *testing.T) {
	userAgent := "Lynx"
	req := httptest.NewRequest("GET", "https://example.com/", nil)
	req.Header.Set("User-Agent", userAgent)
	ev := libhoney.NewEvent()
	AddRequestProps(req, ev)
	fs := ev.Fields()
	assert.Equal(t, userAgent, fs["request.header.user_agent"])
}

func TestXForwardedForHeader(t *testing.T) {
	xForwardedFor := "1.2.3.4"
	req := httptest.NewRequest("GET", "https://example.com/", nil)
	req.Header.Set("X-Forwarded-For", xForwardedFor)
	ev := libhoney.NewEvent()
	AddRequestProps(req, ev)
	fs := ev.Fields()
	assert.Equal(t, xForwardedFor, fs["request.header.x_forwarded_for"])
}

func TestXForwardedProtoHeader(t *testing.T) {
	xForwardedProto := "https"
	req := httptest.NewRequest("GET", "https://example.com/", nil)
	req.Header.Set("X-Forwarded-Proto", xForwardedProto)
	ev := libhoney.NewEvent()
	AddRequestProps(req, ev)
	fs := ev.Fields()
	assert.Equal(t, xForwardedProto, fs["request.header.x_forwarded_proto"])
}
