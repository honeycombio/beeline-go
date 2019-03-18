package hnyhttprouter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	beeline "github.com/honeycombio/beeline-go"
	libhoney "github.com/honeycombio/libhoney-go"
	"github.com/honeycombio/libhoney-go/transmission"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
)

func TestHTTPRouterMiddleware(t *testing.T) {
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
	r, _ := http.NewRequest("GET", "/hello/pooh", nil)
	w := httptest.NewRecorder()

	// build the httprouter mux router with Middleware
	router := httprouter.New()
	router.GET("/hello/:name", Middleware(func(_ http.ResponseWriter, _ *http.Request, _ httprouter.Params) {}))
	// handle the request
	router.ServeHTTP(w, r)

	// verify the MockOutput caught the well formed event
	evs := mo.Events()
	assert.Equal(t, 1, len(evs), "one event is created with one request through the Middleware")
	fields := evs[0].Data
	status, ok := fields["response.status_code"]
	assert.True(t, ok, "'status_code' field must exist on middleware generated event")
	assert.Equal(t, 200, status, "successfully served request should have status 200")
	name, ok := fields["handler.vars.name"]
	assert.True(t, ok, "handler.vars.name field must exist on middleware generated event")
	assert.Equal(t, "pooh", name, "successfully served request should have name var populated")
}

func TestHTTPRouterMiddlewareReturnsStatusCode(t *testing.T) {
	// set up libhoney to catch events instead of send them
	mo := &transmission.MockSender{}
	client, err := libhoney.NewClient(libhoney.ClientConfig{
		APIKey:       "placeholder",
		Dataset:      "placeholder",
		APIHost:      "placeholder",
		Transmission: mo})
	assert.Equal(t, nil, err)
	beeline.Init(beeline.Config{Client: client})

	r, _ := http.NewRequest("GET", "/does_not_exist", nil)
	w := httptest.NewRecorder()

	router := httprouter.New()
	handler := func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.WriteHeader(404)
	}
	router.GET("/does_not_exist", Middleware(handler))
	router.ServeHTTP(w, r)

	evs := mo.Events()
	assert.Equal(t, 1, len(evs), "one event is created with one request through the Middleware")
	fields := evs[0].Data
	status, ok := fields["response.status_code"]
	assert.True(t, ok, "'status_code' field must exist on middleware generated event")
	assert.Equal(t, http.StatusNotFound, status)

}
