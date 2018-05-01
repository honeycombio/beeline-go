package hnyhttprouter_test

import (
	"fmt"
	"log"
	"net/http"

	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/wrappers/hnyhttprouter"
	"github.com/honeycombio/beeline-go/wrappers/hnynethttp"
	"github.com/julienschmidt/httprouter"
)

func main() {
	// Initialize beeline. The only required field is WriteKey.
	beeline.Init(beeline.Config{
		WriteKey: "abcabc123123",
		Dataset:  "sql",
		// for demonstration, send the event to STDOUT intead of Honeycomb.
		// Remove the STDOUT setting when filling in a real write key.
		STDOUT: true,
	})

	router := httprouter.New()

	// call regular httprouter Handles with wrappers to extract parameters
	router.GET("/hello/:name", hnyhttprouter.Middleware(Hello))
	// though the wrapper also works on routes that don't have parameters
	router.GET("/", hnyhttprouter.Middleware(Index))

	// wrap the main router to set everything up for instrumenting
	log.Fatal(http.ListenAndServe(":8080", hnynethttp.WrapHandler(router)))
}

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Welcome!\n")
}

func Hello(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	beeline.AddField(r.Context(), "inHello", true)
	fmt.Fprintf(w, "hello, %s!\n", ps.ByName("name"))
}

// Produces an event like this:
// {
//   "data": {
//     "duration_ms": 0.63284,
//     "handler.name": "main.Hello",
//     "handler.vars.name": "foo",
//     "inHello": true,
//     "meta.localhostname": "cobbler",
//     "meta.type": "http request",
//     "request.content_length": 0,
//     "request.header.user_agent": "curl/7.54.0",
//     "request.host": "",
//     "request.method": "GET",
//     "request.path": "/hello/foo",
//     "request.proto": "HTTP/1.1",
//     "request.remote_addr": "[::1]:52539",
//     "response.status_code": 200
//     "trace.trace_id": "91be396a-41a1-44aa-9f0a-25bf779448cc",
//   },
//   "time": "2018-04-06T22:55:05.040951984-07:00"
// }

// Example http router setup. Run and visit the instrumented routes to generate events.
func Example() {} // This tells godocs that this file is an example.
