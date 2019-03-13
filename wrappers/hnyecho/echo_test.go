package hnyecho

import (
	"net/http"
	"net/http/httptest"
	"testing"

	libhoney "github.com/honeycombio/libhoney-go"
	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"
)

func TestEchoMiddleware(t *testing.T) {
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

	// set up the Echo router with the EchoWrapper middleware
	router := echo.New()
	router.Use(New().Middleware())
	router.GET("/hello/:name", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })
	// handle the request
	router.ServeHTTP(w, r)

	// verify the MockOutput caught the well formed event
	evs := evCatcher.Events()
	assert.Equal(t, 1, len(evs), "one event is created with one request through the Middleware")
	fields := evs[0].Fields()
	// status code
	status, ok := fields["response.status_code"]
	assert.True(t, ok, "response.status_code field must exist on middleware generated event")
	assert.Equal(t, 200, status, "successfully served request should have status 200")
	// response size
	size, ok := fields["response.size"]
	assert.True(t, ok, "response.size field must exist on middleware generated event")
	assert.Equal(t, int64(2), size, "successfully served request should have a response size of 2")
	// path params
	name, ok := fields["app.name"]
	assert.True(t, ok, "app.name field must exist on middleware generated event")
	assert.Equal(t, "pooh", name, "successfully served request should have name var populated")
}
