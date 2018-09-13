package common

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHostHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "beecom.com"
	props := GetRequestProps(req)
	assert.Equal(t, "beecom.com", props["request.host"])
}

func TestNoHostHeader(t *testing.T) {
	// if there is no host header, httptest defaults to using `example.com`
	req := httptest.NewRequest("GET", "/", nil)
	props := GetRequestProps(req)
	assert.Equal(t, "example.com", props["request.host"])
}

func TestURLHostHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "https://doorcom.com/", nil)
	props := GetRequestProps(req)
	assert.Equal(t, "doorcom.com", props["request.host"])
}

func TestUserAgentHeader(t *testing.T) {
	userAgent := "Lynx"
	req := httptest.NewRequest("GET", "https://unused.com/", nil)
	req.Header.Set("User-Agent", userAgent)
	props := GetRequestProps(req)
	assert.Equal(t, userAgent, props["request.header.user_agent"])
}

func TestXForwardedForHeader(t *testing.T) {
	xForwardedFor := "1.2.3.4"
	req := httptest.NewRequest("GET", "https://unused.com/", nil)
	req.Header.Set("X-Forwarded-For", xForwardedFor)
	props := GetRequestProps(req)
	assert.Equal(t, xForwardedFor, props["request.header.x_forwarded_for"])
}

func TestXForwardedProtoHeader(t *testing.T) {
	xForwardedProto := "https"
	req := httptest.NewRequest("GET", "https://unused.com/", nil)
	req.Header.Set("X-Forwarded-Proto", xForwardedProto)
	props := GetRequestProps(req)
	assert.Equal(t, xForwardedProto, props["request.header.x_forwarded_proto"])
}
