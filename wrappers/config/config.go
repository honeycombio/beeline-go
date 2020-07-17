package config

import (
	"net/http"
	"github.com/honeycombio/beeline-go/propagation"
)

// HTTPTraceParserHook is a function that will be invoked on all incoming HTTP requests
// when it is passed as a parameter to an http.Handler wrapper function such as the
// one provided in the hnynethttp package. It can be used to create a PropagationContext
// object using trace context propagation headers in the provided http.Request. It is
// expected that this hook will use one of the unmarshal functions exported in the
// propagation package for a number of supported formats (e.g. Honeycomb, AWS,
// W3C Trace Context, etc).
type HTTPTraceParserHook func(*http.Request) *propagation.PropagationContext

// HTTPTracePropagationHook is a function that will be invoked on all outgoing HTTP requests
// when it is passed as a parameter to a RoundTripper wrapper function such as the one
// provided in the hnynethttp package. It can be used to create a map of header names
// to header values that will be injected in the outgoing request. The information in
// the provided http.Request can be used to make decisions about what headers to include
// in the outgoing request, for example based on the hostname of the target of the request.
// The information in the provided PropagationContext should be used to create the serialized
// header values. It is expected that this hook will use one of the marshal functions exported
// in the propagation package for a number of supported formats (e.g. Honeycomb, AWS,
// W3C Trace Context, etc).
type HTTPTracePropagationHook func(*http.Request, *propagation.PropagationContext) map[string]string

// WraperConfig stores configuration options used by various wrappers.
type WrapperConfig struct {
	HTTPParserHook      HTTPTraceParserHook
	HTTPPropagationHook HTTPTracePropagationHook
}
