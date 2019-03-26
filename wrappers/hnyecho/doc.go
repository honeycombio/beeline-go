// Package hnyecho has middleware to use with the Echo router.
//
// Summary
//
// hnyecho provides Honeycomb instrumentation for the Echo router via middleware.
// It is recommended to put this middleware first in the chain via Echo.Use().
// A Honeycomb event will be generated for every request that comes through your
// Echo router, with basic http fields added. In addition, route related fields will
// be added for that request route.
//
// For a complete example showing this wrapper in use, please see the examples in
// https://github.com/honeycombio/beeline-go/tree/master/examples
//
package hnyecho
