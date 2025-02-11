# Honeycomb Beeline for Go

[![OSS Lifecycle](https://img.shields.io/osslifecycle/honeycombio/beeline-go?color=pink)](https://github.com/honeycombio/home/blob/main/honeycomb-oss-lifecycle-and-practices.md)
[![CircleCI](https://circleci.com/gh/honeycombio/beeline-go.svg?style=shield)](https://circleci.com/gh/honeycombio/beeline-go)
[![GoDoc](https://godoc.org/github.com/honeycombio/beeline-go?status.svg)](https://godoc.org/github.com/honeycombio/beeline-go)

⚠️**STATUS**: This project is being Sunset. See [this issue](https://github.com/honeycombio/beeline-go/issues/449) for more details.

⚠️**Note**: Beelines are Honeycomb's legacy instrumentation libraries. We embrace OpenTelemetry as the effective way to instrument applications. For any new observability efforts, we recommend [instrumenting with OpenTelemetry](https://docs.honeycomb.io/send-data/go/opentelemetry-sdk/).

This package makes it easy to instrument your Go app to send useful events to [Honeycomb](https://www.honeycomb.io), a service for debugging your software in production.
- [Usage and Examples](https://docs.honeycomb.io/getting-data-in/beelines/go-beeline/)
- [API Reference](https://godoc.org/github.com/honeycombio/beeline-go)
  - For each [wrapper](wrappers/), please see the [godoc](https://godoc.org/github.com/honeycombio/beeline-go#pkg-subdirectories)

## Dependencies

Golang 1.19+

## Contributions

Features, bug fixes and other changes to `beeline-go` are gladly accepted. Please
open issues or a pull request with your change. Remember to add your name to the
CONTRIBUTORS file!

All contributions will be released under the Apache License 2.0.
