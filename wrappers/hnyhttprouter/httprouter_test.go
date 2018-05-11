package hnyhttprouter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	libhoney "github.com/honeycombio/libhoney-go"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
)

func TestHTTPRouterMiddleware(t *testing.T) {
	// set up libhoney to catch events instead of send them
	evCatcher := &libhoney.MockOutput{}
	libhoney.Init(libhoney.Config{
		WriteKey: "abcd",
		Dataset:  "efgh",
		Output:   evCatcher,
	})
	// build a sample request to generate an event
	r, _ := http.NewRequest("GET", "/hello/pooh", nil)
	w := httptest.NewRecorder()

	// build the httprouter mux router with Middleware
	router := httprouter.New()
	router.GET("/hello/:name", Middleware(func(_ http.ResponseWriter, _ *http.Request, _ httprouter.Params) {}))
	// handle the request
	router.ServeHTTP(w, r)

	// verify the MockOutput caught the well formed event
	evs := evCatcher.Events()
	assert.Equal(t, 1, len(evs), "one event is created with one request through the Middleware")
	fields := evs[0].Fields()
	status, ok := fields["response.status_code"]
	assert.True(t, ok, "status field must exist on middleware generated event")
	assert.Equal(t, 200, status, "successfully served request should have status 200")
	name, ok := fields["handler.vars.name"]
	assert.True(t, ok, "handler.vars.name field must exist on middleware generated event")
	assert.Equal(t, "pooh", name, "successfully served request should have name var populated")
}
