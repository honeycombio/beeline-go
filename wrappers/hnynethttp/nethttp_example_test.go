package hnynethttp

import (
	"net/http"
)

func ExampleWrapHandler() {
	// assume you have a handler named hello
	var hello func(w http.ResponseWriter, r *http.Request)

	globalmux := http.NewServeMux()
	// add a bunch of routes to the muxer
	globalmux.HandleFunc("/hello/", hello)

	// wrap the globalmux with the honeycomb middleware to send one event per
	// request
	http.ListenAndServe(":8080", WrapHandler(globalmux))
}

func ExampleWrapHandlerFunc() {
	// assume you have a handler function named helloServer
	var helloServer func(w http.ResponseWriter, r *http.Request)

	http.HandleFunc("/hello", WrapHandlerFunc(helloServer))

}
