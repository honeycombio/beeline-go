// package hnygorilla has Middleware to use with the Goji muxer
//
// Summary
//
// hnygorilla has Middleware to wrap individual handlers, and is best used in
// conjunction with the nethttp WrapHandler function. Using these two together
// will get you an event for every request that comes through your application
// while also decorating the most interesting paths (the hantdlers that you
// wrap) with additional fields from the Goji patterns.
//
// See the examples/gorilla for an example of how these two work together
//
package hnygorilla
