package common

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/honeycombio/beeline-go/propagation"
	"github.com/honeycombio/beeline-go/trace"
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

// objForDBCalls gives us an object off which to hang the fake database call
// since the database naming thing looks for actual database calls by name
// it needs a function named as one of the functions in the `dbNames` list
// somewhere in the call stack in order to work. This is essentially a mock
// db call for it to find.
type objForDBCalls struct{}

func (objForDBCalls) ExecContext(bld *libhoney.Builder, query string) *libhoney.Event {
	return sharedDBEvent(bld, query)
}

// TestSharedDBEvent verifies that the name field is set to something
func TestSharedDBEvent(t *testing.T) {
	bld := libhoney.NewBuilder()
	query := "this is sql really promise"
	var ev *libhoney.Event

	// first test uses sharedDBEvent from outside a blessed db path, and it
	// shouldn't really work well but it shouldn't crash
	func() { ev = sharedDBEvent(bld, query) }()
	assert.Equal(t, "db", ev.Fields()["name"], "being called with a non-database call returns default 'db'")
	assert.Equal(t, "", ev.Fields()["db.call"], "being called with a non-database call does not set db.call")
	assert.Equal(t, "TestSharedDBEvent", ev.Fields()["db.caller"], "caller should still be TestSharedDBEvent even if no DB call was found")

	// now we test it as though it really is coming from a DB package with a
	// real DB call like ExecContext. This best models how the instrumentation
	// will be set in a real world use.
	o := objForDBCalls{}
	ev = o.ExecContext(bld, query)
	assert.Equal(t, "ExecContext", ev.Fields()["name"], "being called with a db-specific call returns that string")
	assert.Equal(t, "ExecContext", ev.Fields()["db.call"], "being called with a db-specific call returns that string")
	assert.Equal(t, "TestSharedDBEvent", ev.Fields()["db.caller"], "caller should be this test function")
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

func TestStartSpanOrTraceFromHTTP(t *testing.T) {
	t.Run("when no propagation headers present, starts a new trace", func(t *testing.T) {
		u, _ := url.Parse("https://test.com")
		req := &http.Request{
			Method: "GET",
			URL:    u,
			Header: make(http.Header),
		}
		ctx, _ := StartSpanOrTraceFromHTTP(req)
		traceFromContext := trace.GetTraceFromContext(ctx)
		assert.Equal(t, "", traceFromContext.GetParentID())
	})
	t.Run("when honeycomb propagation header present, uses honeycomb", func(t *testing.T) {
		u, _ := url.Parse("https://test.com")
		header := make(http.Header)
		header.Set(propagation.TracePropagationHTTPHeader, "1;trace_id=abcdef,parent_id=12345")
		req := &http.Request{
			Method: "GET",
			URL:    u,
			Header: header,
		}
		ctx, _ := StartSpanOrTraceFromHTTP(req)
		traceFromContext := trace.GetTraceFromContext(ctx)
		assert.Equal(t, "12345", traceFromContext.GetParentID())
		assert.Equal(t, "abcdef", traceFromContext.GetTraceID())
	})
	t.Run("when w3c propagation header present, uses w3c", func(t *testing.T) {
		u, _ := url.Parse("https://test.com")
		header := make(http.Header)
		header.Set(propagation.TraceparentHeader, "00-7f042f75651d9782dcff93a45fa99be0-c998e73e5420f609-01")
		req := &http.Request{
			Method: "GET",
			URL:    u,
			Header: header,
		}
		ctx, _ := StartSpanOrTraceFromHTTP(req)
		traceFromContext := trace.GetTraceFromContext(ctx)
		assert.Equal(t, "c998e73e5420f609", traceFromContext.GetParentID())
		assert.Equal(t, "7f042f75651d9782dcff93a45fa99be0", traceFromContext.GetTraceID())
	})
	t.Run("when both honeycomb and w3c propagation headers present, uses honeycomb", func(t *testing.T) {
		u, _ := url.Parse("https://test.com")
		header := make(http.Header)
		header.Set(propagation.TracePropagationHTTPHeader, "1;trace_id=abcdef,parent_id=12345")
		header.Set(propagation.TraceparentHeader, "00-7f042f75651d9782dcff93a45fa99be0-c998e73e5420f609-01")
		req := &http.Request{
			Method: "GET",
			URL:    u,
			Header: header,
		}
		ctx, _ := StartSpanOrTraceFromHTTP(req)
		traceFromContext := trace.GetTraceFromContext(ctx)
		assert.Equal(t, "12345", traceFromContext.GetParentID())
		assert.Equal(t, "abcdef", traceFromContext.GetTraceID())
	})
}
