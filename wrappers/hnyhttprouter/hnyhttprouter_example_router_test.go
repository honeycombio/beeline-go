package hnyhttprouter

import (
	"log"
	"net/http"

	"github.com/honeycombio/beeline-go/wrappers/hnynethttp"
	"github.com/julienschmidt/httprouter"
)

func ExampleMiddleware() {
	// assume you have handlers named hello and index
	var hello func(w http.ResponseWriter, r *http.Request, _ httprouter.Params)
	var index func(w http.ResponseWriter, r *http.Request, _ httprouter.Params)

	router := httprouter.New()

	// call regular httprouter Handles with wrappers to extract parameters
	router.GET("/hello/:name", Middleware(hello))
	// though the wrapper also works on routes that don't have parameters
	router.GET("/", Middleware(index))

	// wrap the main router to set everything up for instrumenting
	log.Fatal(http.ListenAndServe(":8080", hnynethttp.WrapHandler(router)))
}
