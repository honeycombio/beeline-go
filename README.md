# Honeycomb Go Magic

[![Build Status](https://travis-ci.org/honeycombio/honeycomb-go-magic.svg?branch=master)](https://travis-ci.org/honeycombio/honeycomb-go-magic)

This package and its subpackages contain bits of code to use to make your life
easier when instrumenting a Go app to send events to Honeycomb. The wrappers
here will collect a handful of useful fields about HTTP requests and SQL calls
in addition to establishing easy patterns to augment this data as your
application runs.

Sign up for a [Honeycomb trial](https://ui.honeycomb.io/signup) to obtain a write key before starting.

# Examples

For each of the wrappers, there is more detailed documentation in that
[wrapper](wrappers/)'s package, and fully functional examples in the
[`examples`](examples/) directory for working examples showing how each of the
different wrappers is used.

# Installation

Add the main package to your `$GOPATH` in the normal way: `go get
github.com/honeycombio/honeycomb-go-magic`. You'll add additional subpackages
depending on the specifics of your application.

# Setup

Regardless of which subpackages are used, there is a small amount of global
configuration to add to your application's startup process. At the bare minimum,
you must pass in your [Team Write Key](https://ui.honeycomb.io/account) and
identify a dataset name to authorize your code to send events to Honeycomb and
tell it where to send events.

```golang
func main() {
	honeycomb.Init(honeycomb.Config{
			WriteKey: "abcabc123123defdef456456",
			Dataset: "myapp",
		})
	...
```

# Use

After initialization, the next step is to find the `http.ListenAndServe` call
and add in the `hnynethttp` wrapper. This establishes the most basic
instrumentation at the outermost layer of the call stack on a per-request basis.

```golang
	http.ListenAndServe(":8080", hnynethttp.WrapHandler(muxer))
```

Once this wrapper is in place, there is a Honeycomb event in the request context
available for use throughout the request's lifecycle.  You could stop here and
have very basic instrumentation, or continue to get additional context.

After the router has parsed the request, more fields specific to that router are
available, such as the specific handler matched or any request parameters that
might be attached to the URL. The wrappers for different HTTP routers handle
this part differently. The next step is to add middleware or specific handler
wrappers in additional places; instructions on how to do this are in each of the
subpackages below.

Available HTTP wrappers:

* [`net/http`](wrappers/hnynethttp)
* [`goji/mux`](wrappers/hnygoji)
* [`gorilla/mux`](wrappers/hnygorilla)
* [`httprouter`](wrappers/hnyhttprouter)

Available DB wrappers:

* [`database/sql`](wrappers/hnysql)
* [`github.com/jmoiron/sqlx`](wrappers/hnysqlx)

## Other HTTP frameworks

If your favorite framework isn't listed here, but supports middleware, look at
the [`hnynethttp`](wrappers/hnynethttp) wrapper. Chances are, a phrase like
["any middleware in the ecosystem that is also compatible with net/http can be
used"](https://github.com/go-chi/chi#middlewares) means that it expects a
function that takes a `http.Handler` and returns a `http.Handler`, which is
exactly what the `WrapHandler` function in `hnynethttp` does.

Try that out and see how far you can get with it and appropriate custom fields.

# Custom Fields

At any time (once the `*http.Request` is decorated with a Honeycomb event) you
can add additional custom fields to the event associated with this request.

```golang
	honeycomb.AddField(req.Context(), "field_name", value)
```

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

# TODO
* write more docs
* add additional http routers and frameworks, eg https://github.com/go-chi/chi
* pull in httpsnoop instead of the existing responseWriter
