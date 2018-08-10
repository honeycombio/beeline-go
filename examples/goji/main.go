package main

import (
	"fmt"
	"net/http"

	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/wrappers/hnygoji"
	"github.com/honeycombio/beeline-go/wrappers/hnynethttp"

	"goji.io"
	"goji.io/pat"
)

func main() {
	// Initialize beeline. The only required field is WriteKey.
	beeline.Init(beeline.Config{
		WriteKey: "abcabc123123",
		Dataset:  "http-goji",
		// for demonstration, send the event to STDOUT instead of Honeycomb.
		// Remove the STDOUT setting when filling in a real write key.
		STDOUT: true,
	})
	// ensure everything gets sent off before we exit
	defer beeline.Close()

	// this example uses a submux just to illustrate the middleware's use
	root := goji.NewMux()
	root.HandleFunc(pat.Get("/hello/:name"), hello)
	root.HandleFunc(pat.Get("/bye/:name"), bye)

	// decorate calls that hit the greetings submux with extra fields
	// this call adds things like the goji pattern to the event
	root.Use(hnygoji.Middleware)

	// wrap the main root handler to get an event out of every request. This
	// gets all the default fields like remote address and status code
	http.ListenAndServe("localhost:8080", hnynethttp.WrapHandler(root))
}

func hello(w http.ResponseWriter, r *http.Request) {
	beeline.AddField(r.Context(), "custom", "in hello")
	name := pat.Param(r, "name") // pat is automatically added to the event
	fmt.Fprintf(w, "Hello, %s!\n", name)
}

func bye(w http.ResponseWriter, r *http.Request) {
	beeline.AddField(r.Context(), "custom", "in bye")
	name := pat.Param(r, "name") // pat is automatically added to the event
	fmt.Fprintf(w, "goodbye, %s!", name)
}

//
// a curl to localhost:8080/hello/ben gets you an event that looks like this:
//
// {
//   "data": {
//     "app.custom": "in hello",
//     "duration_ms": 0.632589,
//     "goji.methods": {
//       "GET": {},
//       "HEAD": {}
//     },
//     "goji.pat": "/hello/:name",
//     "goji.pat.name": "ben",
//     "goji.path_prefix": "/hello/",
//     "handler.name": "main.hello",
//     "handler.type": "http.HandlerFunc",
//     "meta.beeline_version": "0.1.0",
//     "meta.local_hostname": "cobbler",
//     "meta.type": "http",
//     "name": "main.hello",
//     "request.content_length": 0,
//     "request.header.user_agent": "curl/7.54.0",
//     "request.host": "localhost:8080",
//     "request.http_version": "HTTP/1.1",
//     "request.method": "GET",
//     "request.path": "/hello/ben",
//     "request.remote_addr": "127.0.0.1:55532",
//     "response.status_code": 200,
//     "trace.span_id": "8be4e6bc-143d-41e5-9cac-b021444f8998",
//     "trace.trace_id": "70761b4d-078a-4fac-a731-bbe9ef3bf542"
//   },
//   "time": "2018-05-15T23:41:39.121095627-07:00"
// }
