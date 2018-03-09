package hnygoji_test

import (
	"fmt"
	"net/http"

	honeycomb "github.com/honeycombio/honeycomb-go-magic"
	"github.com/honeycombio/honeycomb-go-magic/wrappers/hnygoji"
	"github.com/honeycombio/honeycomb-go-magic/wrappers/hnynethttp"

	"goji.io"
	"goji.io/pat"
)

func main() {
	// Initialize honeycomb. The only required field is WriteKey.
	honeycomb.Init(honeycomb.Config{
		WriteKey: "abcabc123123",
		Dataset:  "http-goji",
		// for demonstration, send the event to STDOUT intead of Honeycomb.
		// Remove the STDOUT setting when filling in a real write key.
		STDOUT: true,
	})

	// this example uses a submux just to illustrate the middleware's use
	root := goji.NewMux()
	greetings := goji.SubMux()
	root.Handle(pat.New("/greet/*"), greetings)
	greetings.HandleFunc(pat.Get("/hello/:name"), hello)
	greetings.HandleFunc(pat.Get("/bye/:name"), bye)

	// decorate calls that hit the greetings submux with extra fields
	greetings.Use(hnygoji.Middleware)

	// wrap the main root handler to get an event out of every request
	http.ListenAndServe("localhost:8080", hnynethttp.WrapHandler(root))
}

func hello(w http.ResponseWriter, r *http.Request) {
	honeycomb.AddField(r.Context(), "custom", "in hello")
	name := pat.Param(r, "name") // pat is automatically added to the event
	fmt.Fprintf(w, "Hello, %s!\n", name)
}

func bye(w http.ResponseWriter, r *http.Request) {
	honeycomb.AddField(r.Context(), "custom", "in bye")
	name := pat.Param(r, "name") // pat is automatically added to the event
	fmt.Fprintf(w, "goodbye, %s!", name)
}

func Example() {} // This tells godocs that this file is an example.
