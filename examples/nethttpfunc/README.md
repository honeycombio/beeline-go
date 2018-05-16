# Example using HTTP and wrapping a HandlerFunc

This example illustrates wrapping a single HnadlerFunc. It is the simplest form
of wrapping an HTTP handler. Though useful for wrapping individual handler
functions, it quickly gets cumbersome when wrapping too many. Wrapping a handler
or a muxer is much more efficient.

This example is runnable with `go run main.go` - it will start listening on port
8080.

Once it's running, in another window, issue a request to the `/hello` endpoint:
`curl localhost:8080/hello` and you should an event appear on STDOUT in the pane
running the example.
