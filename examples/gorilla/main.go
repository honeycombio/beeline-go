package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/wrappers/hnygorilla"
	"github.com/honeycombio/beeline-go/wrappers/hnynethttp"
)

func main() {
	// Initialize beeline. The only required field is WriteKey.
	beeline.Init(beeline.Config{
		WriteKey: "abcabc123123",
		Dataset:  "http-gorilla",
		// for demonstration, send the event to STDOUT intead of Honeycomb.
		// Remove the STDOUT setting when filling in a real write key.
		STDOUT: true,
	})
	// ensure everything gets sent off before we exit
	defer beeline.Close()

	r := mux.NewRouter()
	r.Use(hnygorilla.Middleware)
	// Routes consist of a path and a handler function.
	r.HandleFunc("/", YourHandler)
	r.HandleFunc("/hello/{person}", HelloHandler)

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe("localhost:8080", hnynethttp.WrapHandler(r)))
}

func YourHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Gorilla!\n"))
}

func HelloHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	person := vars["person"]
	beeline.AddField(r.Context(), "inHello", true)
	w.Write([]byte(fmt.Sprintf("Gorilla! Gorilla! %s\n", person)))
}

// generates an event that looks like this:
//
// $ curl localhost:8080/hello/foo
// {
//   "data": {
//     "duration_ms": 0.092819,
//     "gorilla.vars.person": "foo",
//     "handler.fnname": "main.HelloHandler",
//     "handler.name": "",
//     "handler.route": "/hello/{person}",
//     "inHello": true,
//     "meta.localhostname": "cobbler",
//     "meta.type": "http request",
//     "request.content_length": 0,
//     "request.header.user_agent": "curl/7.54.0",
//     "request.host": "localhost:8080",
//     "request.method": "GET",
//     "request.path": "/hello/foo",
//     "request.proto": "HTTP/1.1",
//     "request.remote_addr": "[::1]:51830",
//     "response.status_code": 200
//     "trace.trace_id": "a2ae3280-3b4d-4bb8-828e-b3707e1416f9",
//   },
//   "time": "2018-04-06T22:12:53.440369114-07:00"
// }
