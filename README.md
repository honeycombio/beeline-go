# Honeycomb Beeline for Go

[![Build Status](https://travis-ci.org/honeycombio/beeline-go.svg?branch=master)](https://travis-ci.org/honeycombio/beeline-go)
[![GoDoc](https://godoc.org/github.com/honeycombio/beeline-go?status.svg)](https://godoc.org/github.com/honeycombio/beeline-go)

This package makes it easy to instrument your Go app to send useful events to [Honeycomb](https://www.honeycomb.io), a service for debugging your software in production.
- [Usage and Examples](https://docs.honeycomb.io/getting-data-in/beelines/go-beeline/)
- [API Reference](https://godoc.org/github.com/honeycombio/beeline-go)
  - For each [wrapper](wrappers/), please see the [godoc](https://godoc.org/github.com/honeycombio/beeline-go#pkg-subdirectories)

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

## Contributions

Features, bug fixes and other changes to `beeline-go` are gladly accepted. Please
open issues or a pull request with your change. Remember to add your name to the
CONTRIBUTORS file!

All contributions will be released under the Apache License 2.0.
