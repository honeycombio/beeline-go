package hnynethttp_test

import (
	"io"
	"net/http"

	honeycomb "github.com/honeycombio/honeycomb-go-magic"
	"github.com/honeycombio/honeycomb-go-magic/wrappers/hnynethttp"
)

func main() {
	// Initialize honeycomb. The only required field is WriteKey.
	honeycomb.Init(honeycomb.Config{
		WriteKey: "abcabc123123",
		Dataset:  "http-mux",
		// for demonstration, send the event to STDOUT intead of Honeycomb.
		// Remove the STDOUT setting when filling in a real write key.
		STDOUT: true,
	})

	globalmux := http.NewServeMux()
	globalmux.HandleFunc("/hello/", hello)

	// wrap the globalmux with the honeycomb middleware to send one event per
	// request
	http.ListenAndServe(":8080", hnynethttp.WrapMuxHandler(globalmux))
}

func hello(w http.ResponseWriter, r *http.Request) {
	// Add some custom field to the event
	honeycomb.AddField(r.Context(), "custom", "Wheee")

	io.WriteString(w, "Hello world!")
}

// // Example events created:
// $ curl localhost:8080/hello/foo/bar
// {
//   "data": {
//     "Trace.TraceId": "5279bdc7-fedc-483b-8e4f-a03b4dbb7f27",
//     "custom": "Wheee",
//     "durationMs": 0.809993,
//     "meta.localhostname": "cobbler.local",
//     "meta.type": "http request",
//     "mux.handler.name": "main.hello",
//     "mux.handler.pattern": "/hello/",
//     "mux.handler.type": "http.HandlerFunc",
//     "request.content_length": 0,
//     "request.header.user_agent": "curl/7.54.0",
//     "request.host": "",
//     "request.method": "GET",
//     "request.path": "/hello/foo/bar",
//     "request.proto": "HTTP/1.1",
//     "request.remote_addr": "[::1]:62874",
//     "response.status_code": 200
//   },
//   "time": "2018-04-06T07:23:31.733501961-07:00"
// }
//
// $ curl localhost:8080/hello
// {
//   "data": {
//     "Trace.TraceId": "0344fd2d-a8d0-47c5-9cbd-2b1170e98699",
//     "durationMs": 0.116998,
//     "meta.localhostname": "cobbler.local",
//     "meta.type": "http request",
//     "mux.handler.name": "",
//     "mux.handler.pattern": "/hello/",
//     "mux.handler.type": "*http.redirectHandler",
//     "request.content_length": 0,
//     "request.header.user_agent": "curl/7.54.0",
//     "request.host": "",
//     "request.method": "GET",
//     "request.path": "/hello",
//     "request.proto": "HTTP/1.1",
//     "request.remote_addr": "[::1]:62878",
//     "response.status_code": 301
//   },
//   "time": "2018-04-06T07:23:44.520335853-07:00"
// }
//
// $ curl localhost:8080/hel
// {
//   "data": {
//     "Trace.TraceId": "cf457c21-cfd1-4714-8ced-65f9668f900e",
//     "durationMs": 0.030252,
//     "meta.localhostname": "cobbler.local",
//     "meta.type": "http request",
//     "mux.handler.name": "net/http.NotFound",
//     "mux.handler.pattern": "",
//     "mux.handler.type": "http.HandlerFunc",
//     "request.content_length": 0,
//     "request.header.user_agent": "curl/7.54.0",
//     "request.host": "",
//     "request.method": "GET",
//     "request.path": "/hel",
//     "request.proto": "HTTP/1.1",
//     "request.remote_addr": "[::1]:62883",
//     "response.status_code": 404
//   },
//   "time": "2018-04-06T07:24:16.40206391-07:00"
// }

// Mux wrapper example. Try `curl localhost:8080/hello/` to create an event.
func ExampleMux() {} // This tells godocs that this file is an example.
