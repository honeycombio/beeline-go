/*
Package hnynethttp provides Honeycomb wrappers for net/http Handlers.

Summary

hnynethttp provides wrappers for all the standard `net/http` types: Handler,
HandlerFunc, and ServeMux

See the examples/http-mux/ and examples/http-vanilla folders at the top level of
this repository for sample programs that demonstrate how to use these wrappers.

For best results, wrap the mux passed to http.ListenAndServe - this will get you
an event for every HTTP request handled by the server. The `http-mux` example
demonstrates this approach.

Wrapping individual handlers or HandleFuncs will generate events only for the
endpoints that are wrapped; 404s, for example, will not generate events. See
`http-vanilla` in the example directory for this approach.

*/
package hnynethttp
