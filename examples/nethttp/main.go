package main

import (
	"fmt"
	"io"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"

	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/wrappers/hnynethttp"
)

func main() {
	// Initialize beeline. The only required field is WriteKey.
	beeline.Init(beeline.Config{
		WriteKey: "abcabc123123",
		Dataset:  "http+sql",
		// for demonstration, send the event to STDOUT instead of Honeycomb.
		// Remove the STDOUT setting when filling in a real write key.
		STDOUT: true,
	})

	globalmux := http.NewServeMux()
	globalmux.HandleFunc("/hello/", hello)

	// wrap the globalmux with the honeycomb middleware to send one event per
	// request
	log.Fatal(http.ListenAndServe(":8080", hnynethttp.WrapHandler(globalmux)))
}

func hello(w http.ResponseWriter, r *http.Request) {
	beeline.AddField(r.Context(), "custom", "Wheee")
	// send our response to the caller
	io.WriteString(w, fmt.Sprintf("Hello world!\n"))
}
