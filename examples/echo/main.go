package main

import (
	"fmt"
	"net/http"

	"github.com/jamietsao/beeline-go/wrappers/hnyecho"
	"github.com/labstack/echo"

	"github.com/honeycombio/beeline-go"
)

func main() {
	// Initialize beeline. The only required field is WriteKey.
	beeline.Init(beeline.Config{
		WriteKey: "abcabc123123",
		Dataset:  "http-echo",
		// for demonstration, send the event to STDOUT instead of Honeycomb.
		// Remove the STDOUT setting when filling in a real write key.
		STDOUT: true,
	})
	// ensure everything gets sent off before we exit
	defer beeline.Close()

	// set up Echo router with hnyecho middleware to provide honeycomb instrumentation
	router := echo.New()
	router.Use(hnyecho.New().Middleware())

	// set up routes for hello and bye
	router.GET("/hello/:name", hello)
	router.GET("/bye/:name", bye)

	// start the Echo router (make sure nothing else is running on 8080)
	router.Start(":8080")
}

func hello(c echo.Context) error {
	c.Request().Context()
	beeline.AddField(c.Request().Context(), "custom", "in hello")
	name := c.Param("name")    // path param is added to event
	foo := c.QueryParam("foo") // query param is added to event

	return c.String(http.StatusOK, fmt.Sprintf("Hello, %s! foo = %s\n", name, foo))
}

func bye(c echo.Context) error {
	c.Request().Context()
	beeline.AddField(c.Request().Context(), "custom", "in bye")
	name := c.Param("name")    // path param is added to event
	foo := c.QueryParam("foo") // query param is added to event

	return c.String(http.StatusOK, fmt.Sprintf("Goodbye, %s! foo = %s\n", name, foo))
}

//
// a curl to localhost:8080/hello/ben?foo=bar gets you an event that looks like this:
//
// {
//     "data": {
//         "app.custom": "in hello",
//         "app.foo": "bar",
//         "app.name": "ben",
//         "duration_ms": 0.074091,
//         "handler.name": "main.hello",
//         "meta.beeline_version": "0.3.6",
//         "meta.local_hostname": "cobbler",
//         "meta.span_type": "root",
//         "meta.type": "http_request",
//         "name": "main.hello",
//         "request.content_length": 0,
//         "request.header.user_agent": "curl/7.54.0",
//         "request.host": "localhost:9080",
//         "request.http_version": "HTTP/1.1",
//         "request.method": "GET",
//         "request.path": "/hello/ben",
//         "request.query": "foo=bar",
//         "request.remote_addr": "[::1]:62972",
//         "request.url": "/hello/ben?foo=bar",
//         "response.status_code": 200,
//         "trace.span_id": "44b54980-f31a-4e7c-8259-49a3e8d88448",
//         "trace.trace_id": "725ec7a1-10d3-486b-be31-727fee32921e"
//     },
//     "time": "2019-03-11T23:21:34.55288-07:00",
//     "dataset": "http-echo"
// }
