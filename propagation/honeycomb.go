package propagation

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/honeycombio/beeline-go/trace"
)

const (
	honeycombTracePropagationHTTPHeader = "X-Honeycomb-Trace"
	honeycombTracePropagationVersion    = 1
)

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

// Parse takes the trace header and creates a SpanContext.
func (hc HoneycombHTTPPropagator) Parse(ctx context.Context, header trace.HeaderSupplier) *trace.SpanContext {
	h := header.Get(honeycombTracePropagationHTTPHeader)
	getVer := strings.SplitN(h, ";", 2)
	if getVer[0] == "1" {
		sc, err := hc.parseV1(getVer[1])
		if err == nil {
			return sc
		}
	}
	return nil
}

// parseV1 takes the trace header, stripped of the version
// string, and returns the component parts. Trace ID and Parent ID are both
// required. If either is absent a nil trace header will be returned.
func (HoneycombHTTPPropagator) parseV1(header string) (*trace.SpanContext, error) {
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

// Insert assembles the trace context header and sets the appropriate headers.
func (h HoneycombHTTPPropagator) Insert(ctx context.Context, header trace.HeaderSupplier) {
	sc := trace.GetRemoteSpanContextFromContext(ctx)
	if sc == nil {
		return
	}
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

	s := fmt.Sprintf(
		"%d;trace_id=%s,parent_id=%s,%scontext=%s",
		1,
		sc.TraceID,
		sc.ParentID,
		datasetClause,
		tcB64,
	)
	header.Set(honeycombTracePropagationHTTPHeader, s)
}
