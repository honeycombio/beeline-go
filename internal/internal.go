package internal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/felixge/httpsnoop"
	"github.com/google/uuid"

	"github.com/honeycombio/beeline-go/internal/sample"
	"github.com/honeycombio/beeline-go/timer"
	libhoney "github.com/honeycombio/libhoney-go"
)

const (
	honeyBuilderContextKey = "honeycombBuilderContextKey"
	honeyEventContextKey   = "honeycombEventContextKey"
)

type ResponseWriter struct {
	http.ResponseWriter
	Status int
}

func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: httpsnoop.Wrap(w, httpsnoop.Hooks{}),
	}
}

func (h *ResponseWriter) WriteHeader(statusCode int) {
	h.Status = statusCode
	h.ResponseWriter.WriteHeader(statusCode)
}

func AddRequestProps(req *http.Request, ev *libhoney.Event) {
	userAgent := req.UserAgent()
	xForwardedFor := req.Header.Get("x-forwarded-for")
	xForwardedProto := req.Header.Get("x-forwarded-proto")

	// identify the type of event
	// Add a variety of details about the HTTP request, such as user agent
	// and method, to any created libhoney event.
	ev.AddField("request.method", req.Method)
	ev.AddField("request.path", req.URL.Path)
	ev.AddField("request.host", req.Host)
	ev.AddField("request.http_version", req.Proto)
	ev.AddField("request.content_length", req.ContentLength)
	ev.AddField("request.remote_addr", req.RemoteAddr)
	// add useful header fields if they exist
	if userAgent != "" {
		ev.AddField("request.header.user_agent", userAgent)
	}
	if xForwardedFor != "" {
		ev.AddField("request.header.x_forwarded_for", xForwardedFor)
	}
	if xForwardedProto != "" {
		ev.AddField("request.header.x_forwarded_proto", xForwardedProto)

	}
	// add any AWS trace headers that might be present
	headers := FindTraceHeaders(req)
	reqProps["trace.trace_id"] = headers.TraceID

	// add a span ID
	id, _ := uuid.NewRandom()
	reqProps["trace.span_id"] = id.String()
	return reqProps
}

type HeaderSource int

const (
	HeaderSourceUnknown HeaderSource = iota
	HeaderSourceBeeline
	HeaderSourceAmazon
	HeaderSourceZipkin
	HeaderSourceJaeger
)

type TraceHeader struct {
	Source   HeaderSource
	TraceID  string
	ParentID string
	SpanID   string
}

// FindTraceHeaders parses tracing headers if they exist
//
// Request-Id: abcd-1234-uuid-v4
// X-Amzn-Trace-Id X-Amzn-Trace-Id: Self=1-67891234-12456789abcdef012345678;Root=1-67891233-abcdef012345678912345678;CalledFrom=app
//
// adds all trace IDs to the passed in event, and returns a trace ID if it finds
// one. Request-ID is preferred over the Amazon trace ID. Will generate a UUID
// if it doesn't find any trace IDs.
//
// NOTE that Amazon actually only means for the latter part of the header to be
// the ID - format is version-timestamp-id. For now though (TODO) we treat it as
// the entire string
func FindTraceHeaders(req *http.Request) *TraceHeader {
	headers := &TraceHeader{}
	var traceID string
	awsHeader := req.Header.Get("X-Amzn-Trace-Id")
	if awsHeader != "" {
		headers.Source = HeaderSourceAmazon
		// break into key=val pairs on `;` and add each key=val header
		ids := strings.Split(awsHeader, ";")
		for _, id := range ids {
			keyval := strings.Split(id, "=")
			if len(keyval) != 2 {
				// malformed keyval
				continue
			}
			// ev.AddField("request.header.aws_trace_id."+keyval[0], keyval[1])
			if keyval[0] == "Root" {
				traceID = keyval[1]
			}
		}
	}
	requestID := req.Header.Get("Request-Id")
	if requestID != "" {
		headers.Source = HeaderSourceBeeline
		// ev.AddField("request.header.request_id", requestID)
		traceID = requestID
	}
	if traceID == "" {
		id, _ := uuid.NewRandom()
		traceID = id.String()
	}
	headers.TraceID = traceID
	return headers
}

// // BuildDBEvent tries to bring together most of the things that need to happen
// // for an event to wrap a DB call in bot the sql and sqlx packages. It returns a
// // function which, when called, dispatches the event that it created. This lets
// // it finish a timer around the call automatically.
// func BuildDBEvent(ctx context.Context, bld *libhoney.Builder, query string, args ...interface{}) (*libhoney.Event, func(error)) {
// 	timer := timer.Start()
// 	ev := bld.NewEvent()
// 	fn := func(err error) {
// 		duration := timer.Finish()
// 		// rollup(ctx, ev, duration)
// 		ev.AddField("duration_ms", duration)
// 		if err != nil {
// 			ev.AddField("db.error", err)
// 		}
// 		ev.Metadata, _ = ev.Fields()["name"]
// 		ev.Send()
// 	}
// 	// addTraceID(ctx, ev)

// 	// get the name of the function that called this one. Strip the package and type
// 	pc, _, _, _ := runtime.Caller(1)
// 	callName := runtime.FuncForPC(pc).Name()
// 	callNameChunks := strings.Split(callName, ".")
// 	ev.AddField("db.call", callNameChunks[len(callNameChunks)-1])
// 	ev.AddField("name", callNameChunks[len(callNameChunks)-1])

// 	if query != "" {
// 		ev.AddField("db.query", query)
// 	}
// 	if args != nil {
// 		ev.AddField("db.query_args", args)
// 	}
// 	return ev, fn
// }

// // rollup takes a context that might contain a parent event, the current event,
// // and a duration. It pulls some attributes from the current event in order to
// // add the duration to a summed timer in the parent event.
// func rollup(ctx context.Context, ev *libhoney.Event, dur float64) {
// 	parentEv := beeline.ContextEvent(ctx)
// 	if parentEv == nil {
// 		return
// 	}
// 	// ok now parentEv exists. lets add this to a total duration for the
// 	// meta.type and the specific db call
// 	evFields := ev.Fields()
// 	pvFields := parentEv.Fields()

// 	// only roll up if we have a meta type
// 	metaType, ok := evFields["meta.type"]
// 	if ok {
// 		// make our field names
// 		totalMetaCountKey := fmt.Sprintf("totals.%s.count", metaType)
// 		totalMetaDurKey := fmt.Sprintf("totals.%s.duration_ms", metaType)
// 		// get the existing values or zero if they're missing
// 		totalTypeCount, _ := pvFields[totalMetaCountKey]
// 		totalTypeCountVal, ok := totalTypeCount.(int)
// 		if !ok {
// 			totalTypeCountVal = 0
// 		}
// 		totalTypeDur, _ := pvFields[totalMetaDurKey]
// 		totalTypeDurVal, ok := totalTypeDur.(float64)
// 		if !ok {
// 			totalTypeDurVal = 0
// 		}
// 		// add them to the parent event
// 		parentEv.AddField(totalMetaCountKey, totalTypeCountVal+1)
// 		parentEv.AddField(totalMetaDurKey, totalTypeDurVal+dur)

// 		// if we're a db call, let's roll up the specific call too.
// 		dbCall, ok := evFields["db.call"]
// 		if ok {
// 			// make our field names
// 			totalCallCountKey := fmt.Sprintf("totals.%s.%s.count", metaType, dbCall)
// 			totalCallDurKey := fmt.Sprintf("totals.%s.%s.duration_ms", metaType, dbCall)
// 			// get the existing values or zero if they're missing
// 			totalCallCount, _ := pvFields[totalCallCountKey]
// 			totalCallCountVal, ok := totalCallCount.(int)
// 			if !ok {
// 				totalCallCountVal = 0
// 			}
// 			totalCallDur, _ := pvFields[totalCallDurKey]
// 			totalCallDurVal, ok := totalCallDur.(float64)
// 			if !ok {
// 				totalCallDurVal = 0
// 			}
// 			// add them to the parent event
// 			parentEv.AddField(totalCallCountKey, totalCallCountVal+1)
// 			parentEv.AddField(totalCallDurKey, totalCallDurVal+dur)
// 		}
// 	}
// }

// func addTraceID(ctx context.Context, ev *libhoney.Event) {
// 	// get a transaction ID from the request's event, if it's sitting in context
// 	if parentEv := beeline.ContextEvent(ctx); parentEv != nil {
// 		if id, ok := parentEv.Fields()["trace.trace_id"]; ok {
// 			ev.AddField("trace.trace_id", id)
// 		}
// 		if id, ok := parentEv.Fields()["trace.span_id"]; ok {
// 			ev.AddField("trace.parent_id", id)
// 		}
// 		id, _ := uuid.NewRandom()
// 		ev.AddField("trace.span_id", id.String())
// 	}
// }

// AddField gets the current span and adds the field as is
func AddField(ctx context.Context, key string, val interface{}) {
	span := CurrentSpan(ctx)
	if span != nil {
		if span.ev != nil {
			span.ev.AddField(key, val)
		}
	}
}

type Trace struct {
	// shouldDropSample is true when this trace should be dropped, false when it
	// should be sent.
	shouldDropSample bool
	sampleRate       int

	headers          TraceHeader
	sent             bool // true when the trace is sent, false otherwise.
	openSpans        []*Span
	closedSpans      []*Span
	rollupFields     map[string]float64
	traceLevelFields map[string]interface{}
}

type Span struct {
	timer    timer.Timer
	traceID  string
	spanID   string
	parentID string
	ev       *libhoney.Event
	// idea - indicate here whether it was a wrapper-created span or a custom
	// span, add some extra protections around only beelines being able to close
	// beeline-started spans or something.
}

const honeyTraceContextKey = "honeycombTraceContextKey"

// GetTraceFromContext pulls a trace off the passed in context or returns nil if
// no trace exists.
func GetTraceFromContext(ctx context.Context) *Trace {
	if ctx != nil {
		if trace, ok := ctx.Value(honeyTraceContextKey).(*Trace); ok {
			return trace
		}
	}
	return nil
}

func (t *Trace) SetTraceIDs(traceID, parentID string) {
	if t.shouldDropSample {
		return
	}
	t.headers.TraceID = traceID
	t.headers.ParentID = parentID
	if len(t.openSpans) >= 1 {
		t.openSpans[0].traceID = traceID
		t.openSpans[0].parentID = parentID
	}
}

func (t *Trace) AddField(key string, val interface{}) {
	if t.shouldDropSample {
		return
	}
	if t.traceLevelFields != nil {
		t.traceLevelFields[key] = val
	}
}

func (t *Trace) AddRollupField(key string, val float64) {
	if t.shouldDropSample {
		return
	}
	numSpans := len(t.openSpans)
	if numSpans == 0 {
		return
	}
	t.openSpans[numSpans-1].ev.AddField(key, val)
	t.rollupFields[key] += t.rollupFields[key] + val
}

// EndCurrentSpan returns true when it's closing the last remaining span
func (t *Trace) EndCurrentSpan() (bool, error) {
	if t.shouldDropSample {
		return false, nil
	}
	numSpans := len(t.openSpans)
	if numSpans == 0 {
		return false, errors.New("no open spans")
	}
	span := t.openSpans[numSpans-1]
	// if it doesn't have duration overridden, set it.
	if _, ok := span.ev.Fields()["duration_ms"]; !ok {
		span.ev.AddField("duration_ms", span.timer.Finish())
	}
	// if this is the root span, add the rollup fields
	if numSpans == 1 {
		for key, val := range t.rollupFields {
			rollupKey := fmt.Sprintf("totals.%s", key)
			span.ev.AddField(rollupKey, val)
		}
	}
	t.closedSpans = append(t.closedSpans, span)
	t.openSpans = t.openSpans[:numSpans-1]
	return len(t.openSpans) == 0, nil
}

// TODO change this to return a span to make it easier to handle sampling
func StartAsyncSpan(ctx context.Context) *libhoney.Event {
	sp := CurrentSpan(ctx)
	if sp == nil {
		return libhoney.NewEvent()
	}
	ev := libhoney.NewEvent()
	ev.AddField("trace.trace_id", sp.traceID)
	ev.AddField("trace.parent_id", sp.spanID)
	newSpan, _ := uuid.NewRandom()
	ev.AddField("trace.span_id", newSpan.String())
	return ev
}

// PutTraceInContext takes an existing context and a trace and pushes the trace
// into the context.  It should replace any traces that already exist in the
// context. The returned error will be not nil if a trace already existed.
func PutTraceInContext(ctx context.Context, trace *Trace) (context.Context, error) {
	return context.WithValue(ctx, honeyTraceContextKey, trace), nil

}

// PushSpanOnStack adds a new span to a trace (or creates the trace if none
// exists).
func PushSpanOnStack(ctx context.Context) context.Context {
	trace := GetTraceFromContext(ctx)
	if trace == nil {
		// if we don't have an existing trace, make one and return
		trace = MakeNewTrace("", "")
		ctx, _ = PutTraceInContext(ctx, trace)
		return ctx
	}
	if trace.shouldDropSample {
		return ctx
	}
	currentSpan := trace.openSpans[len(trace.openSpans)-1]
	// make a new span using the parent's span ID as my parent ID
	spanID, _ := uuid.NewRandom()
	span := &Span{
		timer:    timer.Start(),
		traceID:  currentSpan.traceID,
		parentID: currentSpan.spanID,
		spanID:   spanID.String(),
		ev:       libhoney.NewEvent(),
	}
	span.ev.AddField("trace.span_id", span.spanID)
	span.ev.AddField("trace.parent_id", span.parentID)
	span.ev.AddField("trace.trace_id", span.traceID)
	newSpanList := append(trace.openSpans, span)
	trace.openSpans = newSpanList
	ctx, _ = PutTraceInContext(ctx, trace)
	return ctx
}

// EndSpan "closes" the current span by popping it off the open stack and
// putting it on the closed stack. It is not sent in case additional trace level
// fields get added they will still make it onto the closed spans.
func EndSpan(ctx context.Context) {
	trace := GetTraceFromContext(ctx)
	if trace.shouldDropSample {
		return
	}
	finished, err := trace.EndCurrentSpan()
	if err != nil {
		// TODO handle this better
		return
	}
	// if this was the last open span, let's dispatch the trace
	if finished {
		SendTrace(trace)
	}
}

// CurrentSpan gets the outermost span in the list, the currently closest
// surrounding span to the code that's calling it. Returns nil when there are no
// spans or we should drop this trace because of sampling.
func CurrentSpan(ctx context.Context) *Span {
	trace := GetTraceFromContext(ctx)
	if numSpans := len(trace.openSpans); numSpans > 0 {
		return trace.openSpans[numSpans-1]
	}
	return nil
}

func (s *Span) AddField(key string, val interface{}) {
	s.ev.AddField(key, val)
}

// CurrentSpan gets the outermost span in the list, the currently closest
// surrounding span to the code that's calling it. Returns nil when there are no
// spans or we should drop this trace because of sampling.
func CurrentSpanFromTrace(trace *Trace) *Span {
	if numSpans := len(trace.openSpans); numSpans > 0 {
		return trace.openSpans[numSpans-1]
	}
	return nil
}

func MakeNewTrace(traceID, parentID string) *Trace {
	if traceID == "" {
		tid, _ := uuid.NewRandom()
		traceID = tid.String()
		pid, _ := uuid.NewRandom()
		parentID = pid.String()
	} else if parentID == "" {
		pid, _ := uuid.NewRandom()
		parentID = pid.String()
	}
	sid, _ := uuid.NewRandom()
	spanID := sid.String()
	ev := libhoney.NewEvent()
	ev.AddField("trace.span_id", spanID)
	ev.AddField("trace.parent_id", parentID)
	ev.AddField("trace.trace_id", traceID)
	span := &Span{
		timer:    timer.Start(),
		traceID:  traceID,
		spanID:   spanID,
		parentID: parentID,
		ev:       ev,
	}
	shouldDropSample := !sample.GlobalSampler.Sample(traceID)
	if shouldDropSample {
		// if we're not going to send this sample, don't initialize anything.
		// We'll drop everything as it comes in to save computation, storage
		return &Trace{
			shouldDropSample: shouldDropSample,
		}
	}
	return &Trace{
		shouldDropSample: shouldDropSample,
		sampleRate:       sample.GlobalSampler.GetSampleRate(),
		openSpans:        []*Span{span},
		traceLevelFields: make(map[string]interface{}),
		rollupFields:     make(map[string]float64),
	}
}

func SendTrace(trace *Trace) error {
	if trace.shouldDropSample {
		return nil
	}
	// if this trace has already been sent, complain
	if trace.sent == true {
		return errors.New("shouldn't send a trace twice.")
	}
	// if there are any remaining open spans, let's close them.
	if len(trace.openSpans) != 0 {
		for range trace.openSpans {
			trace.EndCurrentSpan()
		}
	}
	for _, span := range trace.closedSpans {
		for k, v := range trace.traceLevelFields {
			span.ev.AddField(k, v)
		}
		span.ev.SampleRate = uint(trace.sampleRate)
		span.ev.SendPresampled()
	}
	trace.sent = true
	return nil
}
