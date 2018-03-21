package main

import (
	"io"
	"log"
	"net/http"

	honeycomb "github.com/honeycombio/honeycomb-go-magic"
)

const writekey = "cf80cea35c40752b299755ad23d2082e"

func main() {
	honeycomb.NewHoneycombInstrumenter(writekey, "")
	http.HandleFunc("/hello", honeycomb.InstrumentHandleFunc(HelloServer))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// hello world, the web server
func HelloServer(w http.ResponseWriter, req *http.Request) {
	honeycomb.AddField(req.Context(), "custom", "Wheee")
	io.WriteString(w, "hello, world!\n")
}

// Original example:
// package main
// import (
// 	"io"
// 	"log"
// 	"net/http"
// )
// func main() {
// 	http.HandleFunc("/hello", HelloServer)
// 	log.Fatal(http.ListenAndServe(":12345", nil))
// }
// // hello world, the web server
// func HelloServer(w http.ResponseWriter, req *http.Request) {
// 	io.WriteString(w, "hello, world!\n")
// }

// Adding lines 8,11,14,21, and modifying 15 yield an event that looks like
// this:
// {
// 	"Timestamp": "2018-03-07 21:42:02.271",
// 	"duration_ms": 0.035626,
// 	"custom": "Wheee",
// 	"handlerName": "main.HelloServer",
// 	"request.content_length": 0,
// 	"request.method": "GET",
// 	"request.path": "/hello",
// 	"request.proto": "HTTP/1.1",
// 	"request.remote_addr": "[::1]:62202",
// 	"request.user_agent": "curl/7.54.0",
// 	"response.status_code": 200,
// }
