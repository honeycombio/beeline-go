# Example using Echo middleware

This example shows off using the hnyecho middleware.  Adding the middleware to the
Echo router using `Echo.Use()` (preferrably as first in the chain) will generate one
Honeycomb event per request.  Fields for basic http properties are added as well as
route related fields (e.g. matched route, path params)

This example is runnable with `go run main.go` - it will start listening on port
8080.

Once it's running, in another window, issue a request to the `/hello` endpoint
with a user's name as the variable: `curl localhost:8080/hello/ben`, and you should
see an event appear on STDOUT in the pane running the example. The event printed will
include a field for the `name` path param.
