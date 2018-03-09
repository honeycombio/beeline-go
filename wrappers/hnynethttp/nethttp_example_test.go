package hnynethttp_test

import (
	"io"
	"log"
	"net/http"

	honeycomb "github.com/honeycombio/honeycomb-go-magic"
	"github.com/honeycombio/honeycomb-go-magic/wrappers/hnynethttp"
)

func main() {
	// Initialize honeycomb. The only required field is WriteKey.
	honeycomb.Init(honeycomb.Config{
		WriteKey: "abcabc123123",
		Dataset:  "http-vanilla",
		// for demonstration, send the event to STDOUT intead of Honeycomb.
		// Remove the STDOUT setting when filling in a real write key.
		STDOUT: true,
	})

	http.HandleFunc("/hello", hnynethttp.WrapHandlerFunc(HelloServer))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// hello world, the web server
func HelloServer(w http.ResponseWriter, req *http.Request) {
	honeycomb.AddField(req.Context(), "custom", "Wheee")
	io.WriteString(w, "hello, world!\n")
}

// Example event created:
// $ go run main.go | jq
// $ curl localhost:8080/hello
// {
//   "data": {
//     "Trace.TraceId": "e18a5d0f-9116-4756-b4bb-4d5e4db1477a",
//     "custom": "Wheee",
//     "durationMs": 0.352607,
//     "handler_func_name": "main.HelloServer",
//     "meta.localhostname": "cobbler.local",
//     "meta.type": "http request",
//     "request.content_length": 0,
//     "request.header.user_agent": "curl/7.54.0",
//     "request.host": "",
//     "request.method": "GET",
//     "request.path": "/hello",
//     "request.proto": "HTTP/1.1",
//     "request.remote_addr": "[::1]:64794",
//     "response.status_code": 200
//   },
//   "time": "2018-04-06T09:48:36.289114189-07:00"
// }

// Simple server example demonstrating how to use `hnynethttp.WrapHandlerFunc(...)`.
// Try `curl localhost:8080/hello` to create an event.
func Example() {} // This tells godocs that this file is an example.
