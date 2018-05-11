package hnynethttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	libhoney "github.com/honeycombio/libhoney-go"
	"github.com/stretchr/testify/assert"
)

func TestWrapHandlerFunc(t *testing.T) {
	// set up libhoney to catch events instead of send them
	evCatcher := &libhoney.MockOutput{}
	libhoney.Init(libhoney.Config{
		WriteKey: "abcd",
		Dataset:  "efgh",
		Output:   evCatcher,
	})
	// build a sample request to generate an event
	r, _ := http.NewRequest("GET", "/hello", nil)
	w := httptest.NewRecorder()

	// build the wrapped handhler on the default mux
	http.HandleFunc("/hello", WrapHandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	// handle the request
	http.DefaultServeMux.ServeHTTP(w, r)

	// verify the MockOutput caught the well formed event
	evs := evCatcher.Events()
	assert.Equal(t, 1, len(evs), "one event is created with one request through the wrapped handler function")
	fields := evs[0].Fields()
	status, ok := fields["response.status_code"]
	assert.True(t, ok, "status field must exist on middleware generated event")
	assert.Equal(t, 200, status, "successfully served request should have status 200")
}

func TestWrapHandler(t *testing.T) {
	// set up libhoney to catch events instead of send them
	evCatcher := &libhoney.MockOutput{}
	libhoney.Init(libhoney.Config{
		WriteKey: "abcd",
		Dataset:  "efgh",
		Output:   evCatcher,
	})
	// build a sample request to generate an event
	r, _ := http.NewRequest("GET", "/hello", nil)
	w := httptest.NewRecorder()

	// build the wrapped handler
	globalmux := http.NewServeMux()
	globalmux.HandleFunc("/hello", func(_ http.ResponseWriter, _ *http.Request) {})
	// handle the request
	WrapHandler(globalmux).ServeHTTP(w, r)

	// verify the MockOutput caught the well formed event
	evs := evCatcher.Events()
	assert.Equal(t, 1, len(evs), "one event is created with one request through the Middleware")
	fields := evs[0].Fields()
	status, ok := fields["response.status_code"]
	assert.True(t, ok, "status field must exist on middleware generated event")
	assert.Equal(t, 200, status, "successfully served request should have status 200")
}
