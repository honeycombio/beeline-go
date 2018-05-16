# Example using HTTP

This example illustrates wrapping a basic net/http mux. It wraps the globalmux
to easily instrument all requests coming into the application.

This example is runnable with `go run main.go` - it will start listening on port
8080.

Once it's running, in another window, issue a request to the `/hello/` endpoint:
`curl localhost:8080/hello/` and you should see several events appear on STDOUT
in the pane running the example.
