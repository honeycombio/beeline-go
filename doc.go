// Package beeline aids adding instrumentation to go apps using Honeycomb.
//
// Summary
//
// This package and its subpackages contain bits of code to use to make your
// life easier when instrumenting a Go app to send events to Honeycomb.
//
// The beeline package provides the entry point - initialization and the basic
// method to add fields to events. Inside the wrappers directory you find
// wrappers for specific HTTP and SQL packages. The standard way to use a
// beeline is to use an HTTP wrapper and then add additional fields as the code
// flows.
//
// The `trace` package offers more direct control over the generated events and
// how they connect together to form traces.
//
// Regardless of which subpackages are used, there is a small amount of global
// configuration to add to your application's startup process. At the bare
// minimum, you must pass in your team write key and identify a dataset name to
// authorize your code to send events to Honeycomb and tell it where to send
// events.
//
//   func main() {
//     beeline.Init(&beeline.Config{
//       WriteKey: "abcabc123123defdef456456",
//       Dataset: "myapp",
//     })
//     ...
//
// Once configured, use one of the subpackages to wrap HTTP handlers and SQL db
// objects.
//
// Examples
//
// There are runnable examples at
// https://github.com/honeycombio/beeline-go/tree/master/examples and examples
// of each wrapper in the godoc.
package beeline
