// Package hnygorilla has Middleware to use with the Gorilla muxer.
//
// Summary
//
// hnygorilla has Middleware to wrap individual handlers, and is best used in
// conjunction with the nethttp WrapHandler function. Using these two together
// will get you an event for every request that comes through your application
// while also decorating the most interesting paths (the handlers that you wrap)
// with additional fields from the Gorilla patterns.
//
// For a complete example showing this wrapper in use, please see the examples in
// https://github.com/honeycombio/beeline-go/tree/master/examples
//
package hnygorilla
