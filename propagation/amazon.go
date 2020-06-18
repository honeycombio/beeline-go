package propagation

import (
	"context"
	"fmt"
	"strings"
	
	"github.com/honeycombio/beeline-go/trace"
)

const amazonTracePropagationHTTPHeader = "X-Amzn-Trace-Id"

// AmazonHTTPPropagator understands how to parse and generate Amazon trace propagation headers
type AmazonHTTPPropagator struct{}

// Parse takes the trace header and creates a SpanContext.
func (AmazonHTTPPropagator) Parse(ctx context.Context, header trace.HeaderSupplier) *trace.SpanContext {
	h := header.Get(amazonTracePropagationHTTPHeader)
	segments := strings.Split(h, ";")

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
	sc := &trace.SpanContext{}
	sc.TraceContext = make(map[string]interface{})
	for _, segment := range segments {
		keyval := strings.SplitN(segment, "=", 2)
		switch strings.ToLower(keyval[0]) {
		case "self":
			sc.ParentID = keyval[1]
		case "root":
			sc.TraceID = keyval[1]
		case "parent":
			sc.ParentID = keyval[1]
		default:
			sc.TraceContext[keyval[0]] = keyval[1]
		}
	}

	// If no header is provided to an ALB or ELB, it will generate a header
	// with a Root field and forwards the request. In this case it should be
	// used as both the parent id and the trace id.
	if sc.TraceID != "" && sc.ParentID == "" {
		sc.ParentID = sc.TraceID
	}

	if sc.TraceID == "" && sc.ParentID != "" {
		return nil
	}

	return sc
}

// Insert assembles the trace context header and sets the appropriate headers.
func (AmazonHTTPPropagator) Insert(ctx context.Context, header trace.HeaderSupplier) {
	sc := trace.GetRemoteSpanContextFromContext(ctx)
	if sc == nil {
		return
	}
	if sc.TraceID == "" || sc.ParentID == "" {
		return
	}
	h := fmt.Sprintf("Root=%s;Parent=%s", sc.TraceID, sc.ParentID)

	// Test if trace context is present, and if so, include it in the header
	if len(sc.TraceContext) != 0 {
		elems := make([]string, len(sc.TraceContext))
		i := 0
		for k, v := range sc.TraceContext {
			elems[i] = fmt.Sprintf("%s=%v", k, v)
			i++
		}
		traceContext := ";" + strings.Join(elems, ";")
		h = h + traceContext
	}

	header.Set(amazonTracePropagationHTTPHeader, h)
}
