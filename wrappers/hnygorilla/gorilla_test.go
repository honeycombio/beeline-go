package hnygorilla

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	beeline "github.com/honeycombio/beeline-go"
	libhoney "github.com/honeycombio/libhoney-go"
	"github.com/honeycombio/libhoney-go/transmission"
	"github.com/stretchr/testify/assert"
)

type testHandler struct{}

func (testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(204)
}

func TestGorillaMiddleware(t *testing.T) {
	// set up libhoney to catch events instead of send them
	mo := &transmission.MockSender{}
	client, err := libhoney.NewClient(libhoney.ClientConfig{
		APIKey:       "placeholder",
		Dataset:      "placeholder",
		APIHost:      "placeholder",
		Transmission: mo})
	assert.Equal(t, nil, err)
	beeline.Init(beeline.Config{Client: client})

	// build the gorilla mux router with Middleware
	router := mux.NewRouter()
	router.Use(Middleware)
	router.HandleFunc("/hello/{name}", func(_ http.ResponseWriter, _ *http.Request) {})
	router.Handle("/yo", testHandler{})

	t.Run("function handler", func(t *testing.T) {
		// build a sample request to generate an event
		r, _ := http.NewRequest("GET", "/hello/pooh", nil)
		w := httptest.NewRecorder()
		// handle the request
		router.ServeHTTP(w, r)

		// verify the MockOutput caught the well formed event
		evs := mo.Events()
		assert.Equal(t, 1, len(evs), "one event is created with one request through the Middleware")
		fields := evs[0].Data
		status, ok := fields["response.status_code"]
		assert.True(t, ok, "status field must exist on middleware generated event")
		assert.Equal(t, 200, status, "successfully served request should have status 200")
		name, ok := fields["gorilla.vars.name"]
		assert.True(t, ok, "gorilla.vars.name field must exist on middleware generated event")
		assert.Equal(t, "pooh", name, "successfully served request should have name var populated")
	})

	t.Run("struct handler should not panic", func(t *testing.T) {
		// build a sample request to generate an event
		r, _ := http.NewRequest("GET", "/yo", nil)
		w := httptest.NewRecorder()
		// handle the request
		router.ServeHTTP(w, r)

		evs := mo.Events()
		assert.Equal(t, 2, len(evs))
		assert.Equal(t, "testHandler", evs[1].Data["name"])
	})
}
