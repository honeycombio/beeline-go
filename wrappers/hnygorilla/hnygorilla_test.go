package hnygorilla

import (
	"net/http"

	"github.com/gorilla/mux"
)

func ExampleMiddleware() {
	// assume you have handlers named root and hello
	var root func(w http.ResponseWriter, r *http.Request)
	var hello func(w http.ResponseWriter, r *http.Request)

	r := mux.NewRouter()
	r.Use(Middleware)
	// Routes consist of a path and a handler function.
	r.HandleFunc("/", root)
	r.HandleFunc("/hello/{person}", hello)
}
