// package beeline aids adding isntrumentation to go apps using Honeycomb.
//
// Summary
//
// This package and its subpackages contain bits of code to use to make your life
// easier when instrumenting a Go app to send events to Honeycomb. The wrappers
// here will collect a handful of useful fields about HTTP requests and SQL calls
// in addition to establishing easy patterns to augment this data as your
// application runs. They are useful for applications handling HTTP requests or
// using the `sql` and `sqlx` packages.
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
// More detail will land here eventually.
package beeline
