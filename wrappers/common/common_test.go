package common

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	libhoney "github.com/honeycombio/libhoney-go"
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

func TestClientReqHostAfterRedirect(t *testing.T) {
	// This test constructs a request the same way the http client constructs
	// one when generating a new request to follow a redirect:
	// https://github.com/golang/go/blob/9baddd3f21230c55f0ad2a10f5f20579dcf0a0bb/src/net/http/client.go#L644-L662
	//
	// When the redirect Location header contains an absolute URL, the new
	// request will have an empty Host field. This ensures we capture a useful
	// request.host property based on the URL itself.
	u, err := url.Parse("http://example.com/")
	assert.NoError(t, err)
	req := &http.Request{
		Method: "GET",
		URL:    u,
		Header: make(http.Header),
	}
	assert.Equal(t, "", req.Host)
	props := GetRequestProps(req)
	assert.Equal(t, "example.com", props["request.host"])
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

// TestSharedDBEvent verifies that the name field is set to something
func TestSharedDBEvent(t *testing.T) {
	bld := libhoney.NewBuilder()
	query := "this is sql really promise"
	// wrap it in another function to get the expected nesting right
	var ev *libhoney.Event
	func() { ev = sharedDBEvent(bld, query) }()
	assert.Equal(t, "TestSharedDBEvent", ev.Fields()["name"], "should get a reasonable name")
}
func TestResponseWriter(t *testing.T) {
	rr := httptest.NewRecorder()
	wr := NewResponseWriter(rr)
	wr.Wrapped.WriteHeader(222)
	assert.Equal(t, 222, wr.Status)
	wr.Wrapped.WriteHeader(333)
	assert.Equal(t, 222, wr.Status)
}

func TestResponseWriterTypeAssertions(t *testing.T) {
	// testResponseWriter implements common http.ResponseWriter optional interfaces
	type testResponseWriter struct {
		http.ResponseWriter
		http.Hijacker
		http.Flusher
		http.CloseNotifier
		http.Pusher
		io.ReaderFrom
	}

	wr := NewResponseWriter(testResponseWriter{})

	if _, ok := interface{}(wr).(http.ResponseWriter); ok {
		t.Errorf("ResponseWriter improperly implements http.ResponseWriter")
	}

	if _, ok := wr.Wrapped.(http.Flusher); !ok {
		t.Errorf("ResponseWriter does not implement http.Flusher")
	}
	if _, ok := wr.Wrapped.(http.CloseNotifier); !ok {
		t.Errorf("ResponseWriter does not implement http.CloseNotifier")
	}
	if _, ok := wr.Wrapped.(http.Hijacker); !ok {
		t.Errorf("ResponseWriter does not implement http.Hijacker")
	}
	if _, ok := wr.Wrapped.(http.Pusher); !ok {
		t.Errorf("ResponseWriter does not implement http.Pusher")
	}
	if _, ok := wr.Wrapped.(io.ReaderFrom); !ok {
		t.Errorf("ResponseWriter does not implement io.ReaderFrom")
	}
}

func TestBuildDBEvent(t *testing.T) {
	b := libhoney.NewBuilder()
	_, sender := BuildDBEvent(b, sql.DBStats{}, "")
	sender(nil)
}

func TestBuildDBSpan(t *testing.T) {
	b := libhoney.NewBuilder()
	ctx := context.Background()
	ctx, _, sender := BuildDBSpan(ctx, b, sql.DBStats{}, "")
	sender(nil)
}
