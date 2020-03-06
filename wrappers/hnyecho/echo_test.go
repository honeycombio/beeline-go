package hnyecho

import (
	"net/http"
	"net/http/httptest"
	"testing"

	beeline "github.com/honeycombio/beeline-go"
	libhoney "github.com/honeycombio/libhoney-go"
	"github.com/honeycombio/libhoney-go/transmission"
	echo "github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestEchoMiddleware(t *testing.T) {
	// set up libhoney to catch events instead of send them
	evCatcher := &transmission.MockSender{}
	client, err := libhoney.NewClient(libhoney.ClientConfig{
		APIKey:       "abcd",
		Dataset:      "efgh",
		APIHost:      "ijkl",
		Transmission: evCatcher,
	})
	assert.Equal(t, nil, err)
	beeline.Init(beeline.Config{Client: client})
	// build a sample request to generate an event
	r, _ := http.NewRequest("GET", "/hello/pooh", nil)
	w := httptest.NewRecorder()

	// set up the Echo router with the EchoWrapper middleware
	router := echo.New()
	router.Use(New().Middleware())
	router.GET("/hello/:name", helloHandler)
	// handle the request
	router.ServeHTTP(w, r)

	// verify the MockOutput caught the well formed event
	evs := evCatcher.Events()
	assert.Equal(t, 1, len(evs), "one event is created with one request through the Middleware")
	fields := evs[0].Data
	// status code
	status, ok := fields["response.status_code"]
	assert.True(t, ok, "response.status_code field must exist on middleware generated event")
	assert.Equal(t, 200, status, "successfully served request should have status 200")
	// response size
	size, ok := fields["response.size"]
	assert.True(t, ok, "response.size field must exist on middleware generated event")
	assert.Equal(t, int64(2), size, "successfully served request should have a response size of 2")
	// handler fields
	handlerNameFields := []string{"handler.name", "name", "route.handler"}
	for _, field := range handlerNameFields {
		handler, ok := fields[field]
		assert.True(t, ok, "handler.name field must exist on middleware generated event")
		assert.Equal(t, "github.com/honeycombio/beeline-go/wrappers/hnyecho.helloHandler", handler, "successfully served request should have correct matched handler")
	}

	// route fields
	route, ok := fields["route"]
	assert.True(t, ok, "route field must exist on middleware generated event")
	assert.Equal(t, "/hello/:name", route, "successfully served request should have matched route")
	name, ok := fields["route.params.name"]
	assert.True(t, ok, "route.params.name field must exist on middleware generated event")
	assert.Equal(t, "pooh", name, "successfully served request should have path param 'name' populated")
}

func helloHandler(c echo.Context) error {
	return c.String(http.StatusOK, "ok")
}
