# Honeycomb Beeline for Go

[![Build Status](https://travis-ci.org/honeycombio/beeline-go.svg?branch=master)](https://travis-ci.org/honeycombio/beeline-go)
[![GoDoc](https://godoc.org/github.com/honeycombio/beeline-go?status.svg)](https://godoc.org/github.com/honeycombio/beeline-go)

This package and its subpackages contain bits of code to use to make your life
easier when instrumenting a Go app to send events to Honeycomb. The wrappers
here will collect a handful of useful fields about HTTP requests and SQL calls
in addition to establishing easy patterns to augment this data as your
application runs.

Documentation and examples are available via [godoc](https://godoc.org/github.com/honeycombio/beeline-go).

Sign up for a [Honeycomb trial](https://ui.honeycomb.io/signup) to obtain a write key before starting.

# Examples

For each of the [wrappers](wrappers/), documentation is found in godoc rather than the github README files. Each package has a fully functional example in the godoc as well, to show all the pieces fit together.

# Installation

Add the main package to your `$GOPATH` in the normal way: `go get
github.com/honeycombio/beeline-go`. You'll add additional subpackages
depending on the specifics of your application.

# Setup

Regardless of which subpackages are used, there is a small amount of global
configuration to add to your application's startup process. At the bare minimum,
you must pass in your [Team Write Key](https://ui.honeycomb.io/account) and
identify a dataset name to authorize your code to send events to Honeycomb and
tell it where to send events.

```golang
import "github.com/honeycombio/beeline-go"
...
func main() {
	beeline.Init(beeline.Config{
			WriteKey: "abcabc123123defdef456456",
			Dataset: "myapp",
		})
	...
```

# Use

After initialization, the next step is to find the `http.ListenAndServe` call
and add in the
[`hnynethttp`](https://godoc.org/github.com/honeycombio/beeline-go/wrappers/hnynethttp)
wrapper. This establishes the most basic instrumentation at the outermost layer
of the call stack on a per-request basis.

```golang
	import "github.com/honeycombio/beeline-go/wrappers/hnynethttp"
	...
	http.ListenAndServe(":8080", hnynethttp.WrapHandler(muxer))
```

Now make a few requests in your web app and open up [Honeycomb](https://ui.honeycomb.io/).
You should see your new dataset! Open it to see the events you just sent.

Once this middleware wrapper is in place, there is a Honeycomb event in the request
context available for use throughout the request's lifecycle.  You could stop here and
have very basic instrumentation, or continue to get additional context.

## Example Questions

* Which endpoints are the slowest?

```
BREAKDOWN: request.path
CALCULATE: P99(duration_ms)
FILTER: meta.type = http request
ORDER BY: P99(duration_ms) DESC
```

* Which endpoints are the most frequently hit?

```
BREAKDOWN: request.path
CALCULATE: COUNT
FILTER: meta.type = http request
```

* Which users are using the endpoint that I'd like to deprecate? (assuming we add a custom field with user.email)

```
BREAKDOWN: app.user.email
CALCULATE: COUNT
FILTER: request.url == <endpoint-url>
```

## Example Event

Depending on which wrappers you use, events will have different fields. This
example gives you a feel for what generated events will look like, though it may
not exactly match what you'll see. All events have at least a timestamp, a
duration, and the `meta.type` field.

```json
{
    "Timestamp": "2018-03-20T00:47:25.339Z",
    "duration_ms": 0.809993,
    "meta.local_hostname": "cobbler.local",
    "meta.type": "http request",
    "mux.handler.name": "main.hello",
    "mux.handler.pattern": "/hello/",
    "mux.handler.type": "http.HandlerFunc",
    "request.content_length": 0,
    "request.header.user_agent": "curl/7.54.0",
    "request.host": "localhost:8080",
    "request.method": "GET",
    "request.path": "/hello/foo/bar",
    "request.proto": "HTTP/1.1",
    "request.remote_addr": "127.0.0.1",
    "response.status_code": 200,
    "trace.trace_id": "5279bdc7-fedc-483b-8e4f-a03b4dbb7f27"
}
```


## Adding Additional Context

At any time (once the `*http.Request` is decorated with a Honeycomb event by the
beeline) you can add additional custom fields to the event associated with this
request.

```golang
beeline.AddField(req.Context(), "field_name", value)
```

By default additional fields you add are namespaced under `app.`, so the field
in the example above would appear in your event as `app.field_name`. This
namespacing groups all your fields together to make them easy to find and
examine.

These additional fields are your opportunity to add important and detailed
context to your instrumentation. Put a timer around a section of code, add per-
user information, include details about what it took to craft a response, and so
on. It is expected that some fields will only be present on some requests. Error
handlers are a great example of this; they will obviously only exist when an
error has occurred.

It is common practice to add in these fields along the way as they are processed
in different levels of middleware.  For example, if you have an authentication
middleware, it would add a field with the authenticated user's ID and name as
soon as it resolves them. Later on in the call stack, you might add additional
fields describing what the user is trying to achieve with this specific HTTP
request.

## Wrappers and Other Middleware

After the router has parsed the request, more fields specific to that router are
available, such as the specific handler matched or any request parameters that
might be attached to the URL. The wrappers for different HTTP routers handle
this part differently. The next step is to add middleware or specific handler
wrappers in additional places; instructions on how to do this are in each of the
subpackages below.

Available HTTP wrappers:

* [`hnynethttp`](https://godoc.org/github.com/honeycombio/beeline-go/wrappers/hnynethttp) (for `net/http`)
* [`hnygoji`](https://godoc.org/github.com/honeycombio/beeline-go/wrappers/hnygoji) (for `goji/mux`)
* [`hnygorilla`](https://godoc.org/github.com/honeycombio/beeline-go/wrappers/hnygorilla) (for `gorilla/mux`)
* [`hnyhttprouter`](https://godoc.org/github.com/honeycombio/beeline-go/wrappers/hnyhttprouter) (for `httprouter`)

Available DB wrappers:

* [`hnysql`](https://godoc.org/github.com/honeycombio/beeline-go/wrappers/hnysql) (for `database/sql`)
* [`hnysqlx`](https://godoc.org/github.com/honeycombio/beeline-go/wrappers/hnysqlx) (for `github.com/jmoiron/sqlx`)
* [`pop`](https://godoc.org/github.com/honeycombio/beeline-go/wrappers/pop) (for `github.com/gobuffalo/pop`)

## Other HTTP Frameworks

If your favorite framework isn't listed here, but supports middleware, look at
the [`hnynethttp`](wrappers/hnynethttp) wrapper. Chances are, a phrase like "any
middleware in the ecosystem that is also compatible with net/http can be
used"([from `go-chi`](https://github.com/go-chi/chi#middlewares)) means that it
expects a function that takes a `http.Handler` and returns a `http.Handler`,
which is exactly what the `WrapHandler` function in `hnynethttp` does.

Try that out and see how far you can get with it and appropriate custom fields.

## Optional Configuration

If you are using both an HTTP wrapper and a SQL package, make sure you pass the
context from the `*http.Request` through to the SQL package using the various
Context-enabled function calls. Doing so will tie the SQL calls back to specific
HTTP requests and you'll get extra fields in your request event showing things
like how much time was spent in the DB, as well as request IDs tying the
separate events together so you can see exactly which DB calls were triggered by
a given event.

For very high throughput services, you can send only a portion of the events
flowing through your service by setting the `SampleRate` during initialization.
This sample rate will send 1/n events, so a sample rate of 5 would send 20% of
all events. For high throughput services, a sample rate of 100 is a good start.

## Troubleshooting

There are two general approaches to finding out what's going wrong when the
beeline isn't doing what you expect.

### The events I'm generating don't contain the content I expect

Use the `STDOUT` flag in configuring the beeline. This will print events to the
terminal *instead* of sending them to Honeycomb. This lets you quickly see
what's getting sent and modify your code to get what you would like to see.

### The events I'm sending aren't being accepted by Honeycomb

Use the `Debug` flag in configuring the beeline. This will print the
responses that come back from Honeycomb to the terminal when sending events.
These responses will have extra detail saying why events are being rejected (or
that they are being accepted) by Honeycomb.

This beeline is also still young, so please reach out to support@honeycomb.io or
ping us with the chat bubble on https://honeycomb.io for assistance.

## Dependencies

The beeline is written with an eye towards a fairly recent set of dependent
packages. Some of the dependencies have changed their API interface over the
past few years (eg goji, uuid, net/http) so you may need to upgrade to get to a
place that works.

* **go 1.9+** - the context package moved into the core library and is used
  extensively by the beeline to make events available to the call stack
* **github.com/google/uuid v0.2+** - the signature for NewRandom started returning
  `UUID, error`
* **github.com/goji/goji v2.0+** - they started using contexts in `net/http` instead
  of their own
