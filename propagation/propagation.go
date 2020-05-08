package propagation

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/honeycombio/beeline-go/trace"
	otelprop "go.opentelemetry.io/otel/api/propagation"
)

const (
	honeycombTracePropagationHTTPHeader = "X-Honeycomb-Trace"
	honeycombTracePropagationVersion    = 1
	amazonTracePropagationHTTPHeader    = "X-Amzn-Trace-Id"
)

type propagationError struct {
	message      string
	wrappedError error
}

func (p *propagationError) Error() string {
	if p.wrappedError == nil {
		return p.message
	}
	return fmt.Sprintf(p.message, p.wrappedError)
}

// assumes a header of the form:

// VERSION;PAYLOAD

// VERSION=1
// =========
// PAYLOAD is a list of comma-separated params (k=v pairs), with no spaces.  recognized
// keys + value types:
//
//  trace_id=${traceId}    - traceId is an opaque ascii string which shall not include ','
//  parent_id=${spanId}    - spanId is an opaque ascii string which shall not include ','
//  dataset=${datasetId}   - datasetId is the slug for the honeycomb dataset to which downstream spans should be sent; shall not include ','
//  context=${contextBlob} - contextBlob is a base64 encoded json object.
//
// ex: X-Honeycomb-Trace: 1;trace_id=weofijwoeifj,parent_id=owefjoweifj,context=SGVsbG8gV29ybGQ=

// HoneycombHTTPPropagator understands how to parse and generate Honeycomb trace propagation headers
type HoneycombHTTPPropagator struct{}

// Extract takes the trace header and creates a SpanContext which is then
// stored in the provided context object.
func (hc HoneycombHTTPPropagator) Extract(ctx context.Context, supplier otelprop.HTTPSupplier) context.Context {
	header := supplier.Get(honeycombTracePropagationHTTPHeader)
	getVer := strings.SplitN(header, ";", 2)
	if getVer[0] == "1" {
		sc, err := hc.extractV1(getVer[1])
		if err == nil {
			ctx = trace.PutRemoteSpanContextInContext(ctx, sc)
		}
	}
	return ctx
}

// extractV1 takes the trace header, stripped of the version
// string, and returns the component parts. Trace ID and Parent ID are both
// required. If either is absent a nil trace header will be returned.
func (HoneycombHTTPPropagator) extractV1(header string) (*trace.SpanContext, error) {
	clauses := strings.Split(header, ",")
	var sc = &trace.SpanContext{}
	var tcB64 string
	for _, clause := range clauses {
		keyval := strings.SplitN(clause, "=", 2)
		switch keyval[0] {
		case "trace_id":
			sc.TraceID = keyval[1]
		case "parent_id":
			sc.ParentID = keyval[1]
		case "dataset":
			sc.Dataset, _ = url.QueryUnescape(keyval[1])
		case "context":
			tcB64 = keyval[1]
		}
	}
	if sc.TraceID == "" && sc.ParentID != "" {
		return nil, &propagationError{"parent_id without trace_id", nil}
	}
	if tcB64 != "" {
		data, err := base64.StdEncoding.DecodeString(tcB64)
		if err != nil {
			return nil, &propagationError{"unable to decode base64 trace context", err}
		}
		sc.TraceContext = make(map[string]interface{})
		err = json.Unmarshal(data, &sc.TraceContext)
		if err != nil {
			return nil, &propagationError{"unable to unmarshal trace context", err}
		}
	}
	return sc, nil

}

// Inject assembles the trace context header and sets it in the supplier.
func (h HoneycombHTTPPropagator) Inject(ctx context.Context, supplier otelprop.HTTPSupplier) {
	sc := trace.GetRemoteSpanContextFromContext(ctx)
	if sc == nil {
		return
	}
	supplier.Set(honeycombTracePropagationHTTPHeader, h.serializeHeader(sc))
}

// SerializeHeader returns a string representation, currently in Honeycomb trace context header format.
func (HoneycombHTTPPropagator) serializeHeader(sc *trace.SpanContext) string {
	tcJSON, err := json.Marshal(sc.TraceContext)
	if err != nil {
		// if we couldn't marshal the trace level fields, leave it to blank
		tcJSON = []byte("")
	}

	tcB64 := base64.StdEncoding.EncodeToString(tcJSON)

	var datasetClause string
	if sc.Dataset != "" {
		datasetClause = fmt.Sprintf("dataset=%s,", url.QueryEscape(sc.Dataset))
	}

	return fmt.Sprintf(
		"%d;trace_id=%s,parent_id=%s,%scontext=%s",
		1,
		sc.TraceID,
		sc.ParentID,
		datasetClause,
		tcB64,
	)
}

// GetAllKeys returns the name of the header used for trace context propagation.
func (HoneycombHTTPPropagator) GetAllKeys() []string {
	return []string{honeycombTracePropagationHTTPHeader}
}

// AmazonHTTPPropagator understands how to parse and generate Amazon trace propagation headers
type AmazonHTTPPropagator struct{}

// Extract takes the trace header and creates a SpanContext which is then stored in the
// provided context.
func (AmazonHTTPPropagator) Extract(ctx context.Context, supplier otelprop.HTTPSupplier) context.Context {
	header := supplier.Get(amazonTracePropagationHTTPHeader)
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
		return ctx
	}

	ctx = trace.PutRemoteSpanContextInContext(ctx, sc)
	return ctx
}

// Inject assembles the trace context header and sets it in the supplier.
func (AmazonHTTPPropagator) Inject(ctx context.Context, supplier otelprop.HTTPSupplier) {
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

	supplier.Set(amazonTracePropagationHTTPHeader, h)
}

// GetAllKeys returns the name of the http header
func (AmazonHTTPPropagator) GetAllKeys() []string {
	return []string{amazonTracePropagationHTTPHeader}
}
