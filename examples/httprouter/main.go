package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	honeycomb "github.com/honeycombio/honeycomb-go-magic"
	"github.com/julienschmidt/httprouter"
)

const writekey = "cf80cea35c40752b299755ad23d2082e"

func main() {
	// initialize Honeycomb instrumentation
	honeycomb.NewHoneycombInstrumenter(writekey, "")

	router := httprouter.New()

	// call regular httprouter Handles with wrappers to extract parameters
	router.GET("/hello/:name", honeycomb.InstrumentHTTPRouterMiddleware(Hello))
	// though the wrapper also works on routes that don't have parameters
	router.GET("/", honeycomb.InstrumentHTTPRouterMiddleware(Index))

	// wrap the main router to set everything up for instrumenting
	log.Fatal(http.ListenAndServe(":8080", honeycomb.InstrumentHandler(router)))
}

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer honeycomb.NewTimer(r.Context(), "Index", time.Now()).Finish()
	fmt.Fprint(w, "Welcome!\n")
}

func Hello(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	honeycomb.AddField(r.Context(), "inHello", true)
	fmt.Fprintf(w, "hello, %s!\n", ps.ByName("name"))
	hnyTimer := honeycomb.NewTimer(r.Context(), "long_hello_job", time.Now())
	time.Sleep(1 * time.Second)
	hnyTimer.Finish()
}

// produces an event like this:
//
// {
//   "data": {
//     "chosenHandle_name": "main.Hello",
//     "duration_ms": 1004.757279,
//     "handler_name": "",
//     "host": "cobbler.local",
//     "inHello": true,
//     "long_hello_job_dur_ms": 1004.501785,
//     "request.content_length": 0,
//     "request.host": "",
//     "request.method": "GET",
//     "request.path": "/hello/foo",
//     "request.proto": "HTTP/1.1",
//     "request.remote_addr": "[::1]:58005",
//     "request.user_agent": "curl/7.54.0",
//     "response.status_code": 200,
//     "vars.name": "foo"
//   },
//   "time": "2018-03-21T14:28:08.57577932-07:00"
// }
// original
// package main
//
// import (
//     "fmt"
//     "github.com/julienschmidt/httprouter"
//     "net/http"
//     "log"
// )
// func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
//     fmt.Fprint(w, "Welcome!\n")
// }
// func Hello(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
//     fmt.Fprintf(w, "hello, %s!\n", ps.ByName("name"))
// }
// func main() {
//     router := httprouter.New()
//     router.GET("/", Index)
//     router.GET("/hello/:name", Hello)
//     log.Fatal(http.ListenAndServe(":8080", router))
// }
