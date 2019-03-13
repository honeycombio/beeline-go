package main

import (
	"fmt"
	"net/http"

	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/wrappers/hnyecho"
	"github.com/labstack/echo"
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
	name := c.Param("name") // path param is added to event

	return c.String(http.StatusOK, fmt.Sprintf("Hello, %s!\n", name))
}

func bye(c echo.Context) error {
	c.Request().Context()
	beeline.AddField(c.Request().Context(), "custom", "in bye")
	name := c.Param("name") // path param is added to event

	return c.String(http.StatusOK, fmt.Sprintf("Goodbye, %s!\n", name))
}

//
// a curl to localhost:8080/hello/ben gets you an event that looks like this:
//
// {
//     "data": {
//         "app.custom": "in hello",
//         "app.name": "ben",
//         "duration_ms": 0.053824,
//         "handler.name": "main.hello",
//         "meta.beeline_version": "0.3.6",
//         "meta.local_hostname": "jamietsao",
//         "meta.span_type": "root",
//         "meta.type": "http_request",
//         "name": "main.hello",
//         "request.content_length": 0,
//         "request.header.user_agent": "curl/7.54.0",
//         "request.host": "localhost:8080",
//         "request.http_version": "HTTP/1.1",
//         "request.method": "GET",
//         "request.path": "/hello/ben",
//         "request.remote_addr": "[::1]:63984",
//         "request.url": "/hello/ben",
//         "response.size": 12,
//         "response.status_code": 200,
//         "trace.span_id": "e4345914-f721-4558-9389-906d484a3f90",
//         "trace.trace_id": "521e55fe-f71f-4c33-ad9a-5e44f366e563"
//     },
//     "time": "2019-03-12T23:38:13.137265-07:00",
//     "dataset": "http-echo"
// }
