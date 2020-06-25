package propagation

import (
	"fmt"
	"strings"
)

const (
	amazonTracePropagationHTTPHeader = "X-Amzn-Trace-Id"
)

// MarshalAmazonTraceContext uses the information in prop to create a trace context header
// in the Amazon X-Ray trace header format. It returns the serialized form of the trace
// context, ready to be inserted into the headers of an outbound HTTP request.
func MarshalAmazonTraceContext(prop *PropagationContext) string {
	if prop == nil {
		return ""
	}

	if prop.TraceID == "" || prop.ParentID == "" {
		return ""
	}

	h := fmt.Sprintf("Root=%s;Parent=%s", prop.TraceID, prop.ParentID)

	if len(prop.TraceContext) != 0 {
		elems := make([]string, len(prop.TraceContext))
		i := 0
		for k, v := range prop.TraceContext {
			elems[i] = fmt.Sprintf("%s=%v", k, v)
			i++
		}
		traceContext := ";" + strings.Join(elems, ";")
		h = h + traceContext
	}

	return h
}

// UnmarshalAmazonTraceContext parses the information provided in header and creates a
// PropagationContext instance.
func UnmarshalAmazonTraceContext(header string) (*PropagationContext, error) {
	segments := strings.Split(header, ";")
	// From https://docs.aws.amazon.com/elasticloadbalancing/latest/application/load-balancer-request-tracing.html
	// If the X-Amzn-Trace-Id header is not present on an incoming request, the load balancer generates a header
	// with a Root field and forwards the request. If the X-Amzn-Trace-Id header is present and has a Root field,
	// the load balancer inserts a Self field, and forwards the request. If an application adds a header with a
	// Root field and a custom field, the load balancer preserves both fields, inserts a Self field, and forwards
	// the request. If the X-Amzn-Trace-Id header is present and has a Self field, the load balancer updates the
	// value of the self field.
	//
	// Using the documentation above (that applies to amazon load balancers) we look for self as the parent id
	// and root as the trace id. In the event that this context comes from a non-load balancer resource (e.g. a
	// service instrumented with an X-Ray SDK) the parent segment ID will be included.
	prop := &PropagationContext{}
	prop.TraceContext = make(map[string]interface{})
	for _, segment := range segments {
		keyval := strings.SplitN(segment, "=", 2)
		switch strings.ToLower(keyval[0]) {
		case "self":
			prop.ParentID = keyval[1]
		case "root":
			prop.TraceID = keyval[1]
		case "parent":
			prop.ParentID = keyval[1]
		default:
			prop.TraceContext[keyval[0]] = keyval[1]
		}
	}

	// If no header is provided to an ALB or ELB, it will generate a header
	// with a Root field and forwards the request. In this case it should be
	// used as both the parent id and the trace id.
	if prop.TraceID != "" && prop.ParentID == "" {
		prop.ParentID = prop.TraceID
	}

	if prop.TraceID == "" && prop.ParentID != "" {
		return nil, &PropagationError{"parent_id without trace_id", nil}
	}

	return prop, nil
}
