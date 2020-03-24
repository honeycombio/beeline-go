package propagation

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
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
//  dataset=${datasetId}   - datasetId is the slug for the honeycomb dataset to which downstream spans should be sent; shall not include ','
//  context=${contextBlob} - contextBlob is a base64 encoded json object.
//
// ex: X-Honeycomb-Trace: 1;trace_id=weofijwoeifj,parent_id=owefjoweifj,context=SGVsbG8gV29ybGQ=

const (
	TracePropagationHTTPHeader = "X-Honeycomb-Trace"
	TracePropagationVersion    = 1
)

type Propagation struct {
	TraceID      string
	ParentID     string
	Dataset      string
	TraceContext map[string]interface{}
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

func MarshalTraceContext(prop *Propagation) string {
	tcJSON, err := json.Marshal(prop.TraceContext)
	if err != nil {
		// if we couldn't marshal the trace level fields, leave it blank
		tcJSON = []byte("")
	}

	tcB64 := base64.StdEncoding.EncodeToString(tcJSON)

	var datasetClause string
	if prop.Dataset != "" {
		datasetClause = fmt.Sprintf("dataset=%s,", url.QueryEscape(prop.Dataset))
	}

	return fmt.Sprintf(
		"%d;trace_id=%s,parent_id=%s,%scontext=%s",
		TracePropagationVersion,
		prop.TraceID,
		prop.ParentID,
		datasetClause,
		tcB64,
	)
}

func UnmarshalTraceContext(header string) (*Propagation, error) {
	// pull the version out of the header
	getVer := strings.SplitN(header, ";", 2)
	if getVer[0] == "1" {
		return UnmarshalTraceContextV1(getVer[1])
	}
	return nil, &PropagationError{fmt.Sprintf("unrecognized version for trace header %s", getVer[0]), nil}
}

// UnmarshalTraceContextV1 takes the trace header, stripped of the version
// string, and returns the component parts. Trace ID and Parent ID are both
// required. If either is absent a nil trace header will be returned.
func UnmarshalTraceContextV1(header string) (*Propagation, error) {
	clauses := strings.Split(header, ",")
	var prop = &Propagation{}
	var tcB64 string
	for _, clause := range clauses {
		keyval := strings.SplitN(clause, "=", 2)
		switch keyval[0] {
		case "trace_id":
			prop.TraceID = keyval[1]
		case "parent_id":
			prop.ParentID = keyval[1]
		case "dataset":
			prop.Dataset, _ = url.QueryUnescape(keyval[1])
		case "context":
			tcB64 = keyval[1]
		}
	}
	if prop.TraceID == "" && prop.ParentID != "" {
		return nil, &PropagationError{"parent_id without trace_id", nil}
	}
	if tcB64 != "" {
		data, err := base64.StdEncoding.DecodeString(tcB64)
		if err != nil {
			return nil, &PropagationError{"unable to decode base64 trace context", err}
		}
		prop.TraceContext = make(map[string]interface{})
		err = json.Unmarshal(data, &prop.TraceContext)
		if err != nil {
			return nil, &PropagationError{"unable to unmarshal trace context", err}
		}
	}
	return prop, nil
}

// UnmarshalAWSTraceContext takes an Amazon ALB or ELB trace header and returns
// the component parts. See
// https://docs.aws.amazon.com/elasticloadbalancing/latest/application/load-balancer-request-tracing.html
// for a description of the AWS trace header format.
func UnmarshalAWSTraceContext(header string) (*Propagation, error) {
	// header will be a semicolon separated list of Field=version-time-id
	// Field will be Root or Parent or something else
	// Root is required, Parent is optional.
	// If Parent is absent, Root is both trace ID and parent span ID
	// if Parent is present, Root is trace ID and Parent is parent span ID
	var prop = &Propagation{}
	fields := strings.Split(header, ";")
	for _, field := range fields {
		nameVal := strings.Split(field, "=")
		if len(nameVal) != 2 {
			// field was not Name=val format. Skip it
			// TODO indicate we've skipped it somehow?
			continue
		}
		name := nameVal[0]
		val := nameVal[1]
		switch name {
		case "Root":
			prop.TraceID = val
		case "Parent":
			prop.ParentID = val
		default:
			// TODO parse other fields here like maybe the beeline can put trace context
			// in the AWS trace ID header as an additional field
		}
	}
	if prop.TraceID == "" {
		// TODO provide more informative errors here
		return nil, &PropagationError{"unable to parse AWS header", nil}
	}
	if prop.ParentID == "" {
		// if Parent was absent, use the trace ID as this trace's parent ID
		prop.ParentID = prop.TraceID
	}
	return prop, nil
}
