package hnynethttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	beeline "github.com/honeycombio/beeline-go"
	libhoney "github.com/honeycombio/libhoney-go"
	"github.com/honeycombio/libhoney-go/transmission"
	"github.com/stretchr/testify/assert"
)

func TestWrapHandlerFunc(t *testing.T) {
	// set up libhoney to catch events instead of send them
	mo := &transmission.MockSender{}
	client, err := libhoney.NewClient(libhoney.ClientConfig{
		APIKey:       "placeholder",
		Dataset:      "placeholder",
		APIHost:      "placeholder",
		Transmission: mo})
	assert.Equal(t, nil, err)
	beeline.Init(beeline.Config{Client: client})
	// build a sample request to generate an event
	r, _ := http.NewRequest("GET", "/hello", nil)
	w := httptest.NewRecorder()

	// build the wrapped handhler on the default mux
	http.HandleFunc("/hello", WrapHandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	http.HandleFunc("/fail", WrapHandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) }))

	// handle successful request
	http.DefaultServeMux.ServeHTTP(w, r)

	// set up + handle failed request
	r, _ = http.NewRequest("GET", "/fail", nil)
	w = httptest.NewRecorder()
	http.DefaultServeMux.ServeHTTP(w, r)

	// verify the MockOutput caught the well formed event
	evs := mo.Events()
	assert.Equal(t, 2, len(evs), "one event is created with one request through the wrapped handler function")
	successfulFields := evs[0].Data
	status, ok := successfulFields["response.status_code"]
	assert.True(t, ok, "status field must exist on middleware generated event")
	assert.Equal(t, 200, status, "successfully served request should have status 200")

	failedFields := evs[1].Data
	status, ok = failedFields["response.status_code"]
	assert.True(t, ok, "status field must exist on middleware generated event")
	assert.Equal(t, http.StatusTeapot, status, "served /fail request should have status 418")
}

func TestWrapHandler(t *testing.T) {
	// set up libhoney to catch events instead of send them
	mo := &transmission.MockSender{}
	client, err := libhoney.NewClient(libhoney.ClientConfig{
		APIKey:       "placeholder",
		Dataset:      "placeholder",
		APIHost:      "placeholder",
		Transmission: mo})
	assert.Equal(t, nil, err)
	beeline.Init(beeline.Config{Client: client})
	// build a sample request to generate an event
	r, _ := http.NewRequest("GET", "/hello", nil)
	w := httptest.NewRecorder()

	// build the wrapped handler
	globalmux := http.NewServeMux()
	globalmux.HandleFunc("/hello", func(_ http.ResponseWriter, _ *http.Request) {})
	globalmux.HandleFunc("/fail", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	// handle the request
	WrapHandler(globalmux).ServeHTTP(w, r)

	// set up + handle failed request
	r, _ = http.NewRequest("GET", "/fail", nil)
	w = httptest.NewRecorder()
	http.DefaultServeMux.ServeHTTP(w, r)

	// verify the MockOutput caught the well formed event
	evs := mo.Events()
	assert.Equal(t, 2, len(evs), "one event is created with one request through the Middleware")
	fields := evs[0].Data
	status, ok := fields["response.status_code"]
	assert.True(t, ok, "status field must exist on middleware generated event")
	assert.Equal(t, 200, status, "successfully served request should have status 200")

	failedFields := evs[1].Data
	status, ok = failedFields["response.status_code"]
	assert.True(t, ok, "status field must exist on middleware generated event")
	assert.Equal(t, http.StatusTeapot, status, "served /fail request should have status 418")
}
