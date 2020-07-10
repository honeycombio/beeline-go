// Package propagation includes types and functions for marshalling and unmarshalling trace
// context headers between various supported formats and an internal representation. It
// provides support for traces that cross process boundaries with support for interoperability
// between various kinds of trace context header formats.
package propagation

import (
	"fmt"
)

// PropagationContext contains information about a trace that can cross process boundaries.
// Typically this information is parsed from an incoming trace context header.
type PropagationContext struct {
	TraceID      string
	ParentID     string
	Dataset      string
	TraceContext map[string]interface{}
	TraceFlags   byte
}

// hasTraceID checks that the trace ID is valid.
func (prop PropagationContext) hasTraceID() bool {
	return prop.TraceID != "" && prop.TraceID != "00000000000000000000000000000000"
}

// hasParentID checks that the parent ID is valid.
func (prop PropagationContext) hasParentID() bool {
	return prop.ParentID != "" && prop.ParentID != "0000000000000000"
}

// IsValid checks if the PropagationContext is valid. A valid PropagationContext has a valid
// trace ID and parent ID.
func (prop PropagationContext) IsValid() bool {
	return prop.hasTraceID() && prop.hasParentID()
}

// Propagation contains information about a trace.
//
// Deprecated: use PropagationContext instead.
type Propagation = PropagationContext

// PropagationError wraps any error encountered while parsing or serializing trace propagation
// contexts.
type PropagationError struct {
	message      string
	wrappedError error
}

// Error returns a formatted message containing the error.
func (p *PropagationError) Error() string {
	if p.wrappedError == nil {
		return p.message
	}
	return fmt.Sprintf(p.message, p.wrappedError)
}

// MarshalTraceContext wraps MarshalHoneycombTraceContext for backwards compatibility.
//
// Deprecated: Use MarshalHoneycombTraceContext.
func MarshalTraceContext(prop *PropagationContext) string {
	return MarshalHoneycombTraceContext(prop)
}

// UnmarshalTraceContext wraps UnmarshalHoneycombTraceContext for backwards compatibility.
//
// Deprecated: Use UnmarshalHoneycombTraceContext
func UnmarshalTraceContext(header string) (*PropagationContext, error) {
	return UnmarshalHoneycombTraceContext(header)
}

// UnmarshalTraceContextV1 wraps UnmarshalHoneycombTraceContextV1 for backwards compatibility.
//
// Deprecated: Use UnmarshalHoneycombTraceContext. Do not call this function directly.
func UnmarshalTraceContextV1(header string) (*PropagationContext, error) {
	return unmarshalHoneycombTraceContextV1(header)
}
