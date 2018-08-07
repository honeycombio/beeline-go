package internal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
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
//  context=${contextBlob} - contextBlob is a base64 encoded json object.
//
// ex: X-Honeycomb-Trace: 1;trace_id=weofijwoeifj,parent_id=owefjoweifj,context=SGVsbG8gV29ybGQ=

const (
	TracePropagationHTTPHeader = "X-Honeycomb-Trace"
	TracePropagationVersion    = 1
)

type Propagation struct {
	TraceHeader
	TraceContext       map[string]interface{}
	TraceContextBase64 string
}

type PropagationError struct {
	message      string
	wrappedError error
}

func (p *PropagationError) Error() string {
	if p.wrappedError == nil {
		return p.message
	}
	return fmt.Sprintf(p.message, p.wrappedError)
}

func MarshalTraceContext(ctx context.Context) string {
	trace := GetTraceFromContext(ctx)
	currentSpan := trace.openSpans[len(trace.openSpans)-1]

	var prop = &Propagation{}
	prop.Source = HeaderSourceBeeline
	prop.TraceID = trace.headers.TraceID
	prop.ParentID = currentSpan.spanID
	prop.TraceContext = trace.traceLevelFields

	tcJSON, err := json.Marshal(prop.TraceContext)
	if err != nil {
		// if we couldn't marshal the trace level fields, leave it blank
		tcJSON = []byte("")
	}

	tcB64 := base64.StdEncoding.EncodeToString(tcJSON)

	return fmt.Sprintf("%d;trace_id=%s,parent_id=%s,context=%s",
		TracePropagationVersion, prop.TraceID, prop.ParentID, tcB64)
}

func UnmarshalTraceContext(header string) (*TraceHeader, map[string]interface{}, error) {
	// pull the version out of the header
	getVer := strings.SplitN(header, ";", 2)
	if getVer[0] == "1" {
		return UnmarshalTraceContextV1(getVer[1])
	}
	return nil, nil, &PropagationError{fmt.Sprintf("unrecognized version for trace header %s", getVer[0]), nil}
}

// UnmarshalTraceContextV1 takes the trace header, stripped of the version
// string, and returns the component parts. Trace ID and Parent ID are both
// required. If either is absent a nil trace header will be returned.
func UnmarshalTraceContextV1(header string) (*TraceHeader, map[string]interface{}, error) {
	clauses := strings.Split(header, ",")
	var prop = &Propagation{}
	prop.Source = HeaderSourceBeeline
	for _, clause := range clauses {
		keyval := strings.SplitN(clause, "=", 2)
		switch keyval[0] {
		case "trace_id":
			prop.TraceID = keyval[1]
		case "parent_id":
			prop.ParentID = keyval[1]
		case "context":
			prop.TraceContextBase64 = keyval[1]
		}
	}
	if prop.TraceID == "" {
		return nil, nil, &PropagationError{"missing trace ID", nil}
	}
	if prop.ParentID == "" {
		return nil, nil, &PropagationError{"missing parent ID", nil}
	}
	if prop.TraceContextBase64 != "" {
		data, err := base64.StdEncoding.DecodeString(prop.TraceContextBase64)
		if err != nil {
			return nil, nil, &PropagationError{"unable to decode base64 trace context", err}
		}
		prop.TraceContext = make(map[string]interface{})
		err = json.Unmarshal(data, &prop.TraceContext)
		if err != nil {
			return nil, nil, &PropagationError{"unable to unmarshal trace context", err}
		}
	}
	return &prop.TraceHeader, prop.TraceContext, nil
}
