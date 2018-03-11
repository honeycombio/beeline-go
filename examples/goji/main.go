package main

import (
	"fmt"
	"net/http"

	honeycomb "github.com/honeycombio/honeycomb-go-magic"
	"goji.io"
	"goji.io/pat"
)

const writekey = "cf80cea35c40752b299755ad23d2082e"

func hello(w http.ResponseWriter, r *http.Request) {
	honeycomb.AddField(r.Context(), "custom", "Wheee")
	name := pat.Param(r, "name")         // pat is automatically added to the event
	relation := pat.Param(r, "relation") // pat is automatically added to the event
	fmt.Fprintf(w, "Hello, my %s %s!", relation, name)
}

func bye(w http.ResponseWriter, r *http.Request) {
	honeycomb.AddField(r.Context(), "custom", "Wheee")
	name := pat.Param(r, "name") // pat is automatically added to the event
	fmt.Fprintf(w, "goodbye, %s!", name)
}

func register(w http.ResponseWriter, r *http.Request) {
	honeycomb.AddField(r.Context(), "custom", "Wheee")
	name := pat.Param(r, "name") // pat is automatically added to the event
	fmt.Fprintf(w, "regging, %s!", name)
}

func deregister(w http.ResponseWriter, r *http.Request) {
	honeycomb.AddField(r.Context(), "custom", "Wheee")
	name := pat.Param(r, "name") // pat is automatically added to the event
	fmt.Fprintf(w, "dregging, %s!", name)
}

func main() {
	hi := honeycomb.NewInstrumenter(writekey)
	root := goji.NewMux()
	greetings := goji.SubMux()
	registrations := goji.SubMux()
	root.Handle(pat.New("/greet/*"), greetings)
	root.Handle(pat.New("/reg/*"), registrations)
	greetings.HandleFunc(pat.Get("/hello/:name/:relation"), hello)
	greetings.HandleFunc(pat.Get("/bye/:name"), bye)
	registrations.HandleFunc(pat.Get("/register/:name"), register)
	registrations.HandleFunc(pat.Get("/deregister/:name"), deregister)

	greetings.Use(hi.InstrumentGojiMiddleware)
	registrations.Use(hi.InstrumentGojiMiddleware)
	http.ListenAndServe("localhost:8000", root)
	// http.ListenAndServe("localhost:8000", root)
}

// root := NewMux()
// users := SubMux()
// root.Handle(pat.New("/users/*"), users)
// albums := SubMux()
// root.Handle(pat.New("/albums/*"), albums)

// // e.g., GET /users/carl
// users.Handle(pat.Get("/:name"), renderProfile)
// // e.g., POST /albums/
// albums.Handle(pat.Post("/"), newAlbum)
