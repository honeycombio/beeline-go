package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	honeycomb "github.com/honeycombio/honeycomb-go-magic"
)

const writekey = "cf80cea35c40752b299755ad23d2082e"

func main() {
	honeycomb.NewHoneycombInstrumenter(writekey, "")
	r := mux.NewRouter()
	r.Use(honeycomb.AddGorillaMiddleware)
	// set a default handler
	r.NotFoundHandler = http.NotFoundHandler()
	// Routes consist of a path and a handler function.
	r.HandleFunc("/", YourHandler)
	r.HandleFunc("/hello/{person}", HelloHandler)

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8080", r))
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

// TODO 404s don't trip the middleware. explicitly wrap the Not Found handler?

// original
//
// func YourHandler(w http.ResponseWriter, r *http.Request) {
// 	w.Write([]byte("Gorilla!\n"))
// }
//
// func HelloHandler(w http.ResponseWriter, r *http.Request) {
// 	vars := mux.Vars(r)
// 	person := vars["person"]
// 	w.Write([]byte(fmt.Sprintf("Gorilla! Gorilla! %s\n", person)))
// }
//
// func main() {
// 	r := mux.NewRouter()
// 	// Routes consist of a path and a handler function.
// 	r.HandleFunc("/", YourHandler)
// 	r.HandleFunc("/hello/{person}", HelloHandler)
//
// 	// Bind to a port and pass our router in
// 	log.Fatal(http.ListenAndServe(":8000", r))
// }
//
//
// generates an event that looks like this:
//
// {
//   "data": {
//     "chosenHandler_name": "main.YourHandler",
//     "durationMs": 0.034328,
//     "gorilla.routeMatched": "/",
//     "host": "cobbler.local",
//     "request.content_length": 0,
//     "request.host": "",
//     "request.method": "GET",
//     "request.path": "/",
//     "request.proto": "HTTP/1.1",
//     "request.remote_addr": "[::1]:63524",
//     "request.user_agent": "curl/7.54.0",
//     "response.status_code": 200
//   },
//   "time": "2018-03-20T12:30:06.140980902-07:00"
// }
