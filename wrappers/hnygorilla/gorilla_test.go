package hnygorilla_test

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	honeycomb "github.com/honeycombio/honeycomb-go-magic"
	"github.com/honeycombio/honeycomb-go-magic/wrappers/hnygorilla"
	"github.com/honeycombio/honeycomb-go-magic/wrappers/hnynethttp"
)

const writekey = "cf80cea35c40752b299755ad23d2082e"

func main() {
	// Initialize honeycomb. The only required field is WriteKey.
	honeycomb.Init(honeycomb.Config{
		WriteKey: "abcabc123123",
		Dataset:  "http-gorilla",
		// for demonstration, send the event to STDOUT intead of Honeycomb.
		// Remove the STDOUT setting when filling in a real write key.
		STDOUT: true,
	})

	r := mux.NewRouter()
	r.Use(hnygorilla.Middleware)
	// Routes consist of a path and a handler function.
	r.HandleFunc("/", YourHandler)
	r.HandleFunc("/hello/{person}", HelloHandler)

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8080", hnynethttp.WrapHandler(r)))
}

func YourHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Gorilla!\n"))
}

func HelloHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	person := vars["person"]
	honeycomb.AddField(r.Context(), "inHello", true)
	w.Write([]byte(fmt.Sprintf("Gorilla! Gorilla! %s\n", person)))
}

// generates an event that looks like this:
//
// $ curl localhost:8080/hello/foo
// {
//   "data": {
//     "Trace.TraceId": "a2ae3280-3b4d-4bb8-828e-b3707e1416f9",
//     "durationMs": 0.092819,
//     "gorilla.vars.person": "foo",
//     "handler.fnname": "main.HelloHandler",
//     "handler.name": "",
//     "handler.route": "/hello/{person}",
//     "inHello": true,
//     "meta.localhostname": "cobbler",
//     "meta.type": "http request",
//     "request.content_length": 0,
//     "request.header.user_agent": "curl/7.54.0",
//     "request.host": "",
//     "request.method": "GET",
//     "request.path": "/hello/foo",
//     "request.proto": "HTTP/1.1",
//     "request.remote_addr": "[::1]:51830",
//     "response.status_code": 200
//   },
//   "time": "2018-04-06T22:12:53.440369114-07:00"
// }

func Example() {} // This tells godocs that this file is an example.
