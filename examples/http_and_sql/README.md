# Example using both HTTP and SQL

This example illustrates what happens when you use multiple wrappers -
specifically one from the HTTP layer accepting incoming requests and one
wrapping DB access. Not only do you get one event for the incoming request and
an event for each DB call, but (because the context is passed all the way down)
the events are connected using a Trace ID and the outer HTTP event gets a few
extra fields indicating how much total time was spent in the DB *for that
request*.

This example is runnable with `go run main.go` - it will start listening on port
8080.

Once it's running, in another window, issue a request to the `/hello/` endpoint:
`curl localhost:8080/hello/` and you should see several events appear on STDOUT
in the pane running the example.
