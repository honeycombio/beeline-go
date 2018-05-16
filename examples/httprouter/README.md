# Example using httprouter middleware

This example shows off using a combination of packages.  Using the
`hnynethttp.WrapHandler` around the main httprouter router gets you one basic
event for every request that comes through, regardless of what handler it hits.
Adding the middleware around each handler gets additional fields that are custom
to a matched route.

This example is runnable with `go run main.go` - it will start listening on port
8080.

Once it's running, in another window, issue a request to the `/hello` endpoint
with a user's name as the variable: `curl localhost:8080/hello/ben` and you
should an event appear on STDOUT in the pane running the example. The event
printed will include the pattern matched (`/hello/:name`) as well as the
contents of the `name` variable.
