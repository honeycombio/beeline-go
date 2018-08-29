package main

import (
	"io"
	"log"
	"net/http"

	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/wrappers/hnynethttp"
)

// Simple server example demonstrating how to use `hnynethttp.WrapHandlerFunc(...)`.
// Try `curl localhost:8080/hello` to create an event.
func main() {

	// Initialize beeline. The only required field is WriteKey.
	beeline.Init(beeline.Config{
		WriteKey: "abcabc123123",
		Dataset:  "http-vanilla",
		// for demonstration, send the event to STDOUT instead of Honeycomb.
		// Remove the STDOUT setting when filling in a real write key.
		STDOUT: true,
	})

	// here only the /hello handler is wrapped, no other endpoints.
	http.HandleFunc("/hello", hnynethttp.WrapHandlerFunc(helloServer))
	log.Fatal(http.ListenAndServe("localhost:8080", nil))

}

// hello world, the web server
func helloServer(w http.ResponseWriter, req *http.Request) {
	beeline.AddField(req.Context(), "custom", "Wheee")
	io.WriteString(w, "hello, world!\n")
}

// Example event created:
// $ go run main.go | jq
// $ curl localhost:8080/hello
// {
//   "data": {
//     "app.custom": "Wheee",
//     "duration_ms": 0.352607,
//     "handler_func_name": "main.HelloServer",
//     "meta.localhostname": "cobbler.local",
//     "meta.type": "http request",
//     "request.content_length": 0,
//     "request.header.user_agent": "curl/7.54.0",
//     "request.host": "localhost:8080",
//     "request.method": "GET",
//     "request.path": "/hello",
//     "request.proto": "HTTP/1.1",
//     "request.remote_addr": "[::1]:64794",
//     "response.status_code": 200
//     "trace.trace_id": "e18a5d0f-9116-4756-b4bb-4d5e4db1477a",
//   },
//   "time": "2018-04-06T09:48:36.289114189-07:00"
// }
