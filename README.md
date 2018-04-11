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

Subpackages are available for some common HTTP routers and two SQL packages. The specifics of how to instrument your code vary depending on which HTTP router you're using. Additional details are below.

Available HTTP wrappers:

* [`net/http`](wrappers/hnynethttp) contains a wrapper that conforms to the `http.Handler` pattern, so is useful when a more specific match is missing
* [`goji/mux`](wrappers/hnygoji)
* [`gorilla/mux`](wrappers/hnygorilla)
* [`httprouter`](wrappers/hnyhttprouter)

Available DB wrappers:

* [`database/sql`](wrappers/hnysql)
* [`github.com/jmoiron/sqlx`](wrappers/hnysqlx)


# TODO
* write more docs
* add additional http routers and frameworks, eg https://github.com/go-chi/chi
* pull in httpsnoop instead of the existing responseWriter
