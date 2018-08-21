package internal

// // FindTraceHeaders parses tracing headers if they exist. Uses beeline headers
// // first, then looks for others.
// //
// // Request-Id: abcd-1234-uuid-v4
// // X-Amzn-Trace-Id X-Amzn-Trace-Id: Self=1-67891234-12456789abcdef012345678;Root=1-67891233-abcdef012345678912345678;CalledFrom=app
// //
// // adds all trace IDs to the passed in event, and returns a trace ID if it finds
// // one. Request-ID is preferred over the Amazon trace ID. Will generate a UUID
// // if it doesn't find any trace IDs.
// //
// // NOTE that Amazon actually only means for the latter part of the header to be
// // the ID - format is version-timestamp-id. For now though (TODO) we treat it as
// // the entire string
// //
// // If getting trace context from another beeline, also returns any fields
// // included to be added to the trace as Trace level fields
// func FindTraceHeaders(req *http.Request) (*TraceHeader, map[string]interface{}, error) {
// 	beelineHeader := req.Header.Get(TracePropagationHTTPHeader)
// 	if beelineHeader != "" {
// 		return UnmarshalTraceContext(beelineHeader)
// 	}
// 	// didn't find it from a beeline, let's go looking elsewhere
// 	headers := &TraceHeader{}
// 	var traceID string
// 	awsHeader := req.Header.Get("X-Amzn-Trace-Id")
// 	if awsHeader != "" {
// 		headers.Source = HeaderSourceAmazon
// 		// break into key=val pairs on `;` and add each key=val header
// 		ids := strings.Split(awsHeader, ";")
// 		for _, id := range ids {
// 			keyval := strings.Split(id, "=")
// 			if len(keyval) != 2 {
// 				// malformed keyval
// 				continue
// 			}
// 			// ev.AddField("request.header.aws_trace_id."+keyval[0], keyval[1])
// 			if keyval[0] == "Root" {
// 				traceID = keyval[1]
// 			}
// 		}
// 	}
// 	requestID := req.Header.Get("Request-Id")
// 	if requestID != "" {
// 		headers.Source = HeaderSourceBeeline
// 		// ev.AddField("request.header.request_id", requestID)
// 		traceID = requestID
// 	}
// 	if traceID == "" {
// 		id, _ := uuid.NewRandom()
// 		traceID = id.String()
// 	}
// 	headers.TraceID = traceID
// 	return headers, nil, nil
// }
