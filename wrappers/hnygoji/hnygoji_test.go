package hnygoji

import (
	"net/http"

	"goji.io"
	"goji.io/pat"
)

func ExampleMiddleware() {
	// assume you have handlers for hello and bye
	var hello func(w http.ResponseWriter, r *http.Request)
	var bye func(w http.ResponseWriter, r *http.Request)

	// this example uses a submux just to illustrate the middleware's use
	root := goji.NewMux()
	greetings := goji.SubMux()
	root.Handle(pat.New("/greet/*"), greetings)
	greetings.HandleFunc(pat.Get("/hello/:name"), hello)
	greetings.HandleFunc(pat.Get("/bye/:name"), bye)

	// decorate calls that hit the greetings submux with extra fields
	greetings.Use(Middleware)
}
