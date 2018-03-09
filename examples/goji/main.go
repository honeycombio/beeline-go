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
	name := pat.Param(r, "name") // pat is automatically added to the event
	fmt.Fprintf(w, "Hello, %s!", name)
}

func bye(w http.ResponseWriter, r *http.Request) {
	honeycomb.AddField(r.Context(), "custom", "Wheee")
	name := pat.Param(r, "name") // pat is automatically added to the event
	fmt.Fprintf(w, "Hello, %s!", name)
}

func register(w http.ResponseWriter, r *http.Request) {
	honeycomb.AddField(r.Context(), "custom", "Wheee")
	name := pat.Param(r, "name") // pat is automatically added to the event
	fmt.Fprintf(w, "Hello, %s!", name)
}

func deregister(w http.ResponseWriter, r *http.Request) {
	honeycomb.AddField(r.Context(), "custom", "Wheee")
	name := pat.Param(r, "name") // pat is automatically added to the event
	fmt.Fprintf(w, "Hello, %s!", name)
}

func main() {
	hi := honeycomb.NewInstrumenter(writekey)
	root := goji.NewMux()
	greetings := goji.SubMux()
	registrations := goji.SubMux()
	greetings.HandleFunc(pat.Get("/hello/:name"), hello)

	http.ListenAndServe("localhost:8000", hi.InstrumentMuxHandler(greetings))
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
