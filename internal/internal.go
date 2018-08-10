package internal

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/honeycombio/beeline-go/internal/sample"
	"github.com/honeycombio/beeline-go/timer"
	libhoney "github.com/honeycombio/libhoney-go"
)

var GlobalConfig Config

type Config struct {
	SamplerHook func(map[string]interface{}) (bool, int)
	PresendHook func(map[string]interface{})
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
// TODO give spans a pointer back to the trace they're in so that you can end a
// specific span rather than only the current one. Necessary to allow concurrent
// use of a trace.
type Span struct {
	shouldDrop bool // used for sampler hook
	timer      timer.Timer
	traceID    string
	spanID     string
	parentID   string
	ev         *libhoney.Event
	// idea - indicate here whether it was a wrapper-created span or a custom
	// span, add some extra protections around only beelines being able to close
	// beeline-started spans or something.
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

// AddField gets the current span and adds the field as is - it does not give
// the field a prefix in the way the public beeline API does. This is necessary
// to add protected fields such as `name` or `duration_ms`
func AddField(ctx context.Context, key string, val interface{}) {
	span := CurrentSpan(ctx)
	if span != nil {
		if span.ev != nil {
			span.ev.AddField(key, val)
		}
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
	prev := t.rollupFields[key]
	t.rollupFields[key] += val
	fmt.Printf("adding %f to rollup field %s prev %f cur %f\n", val, key, prev, t.rollupFields[key])
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
func StartAsyncSpan(ctx context.Context, name string) *libhoney.Event {
	sp := CurrentSpan(ctx)
	if sp == nil {
		return libhoney.NewEvent()
	}
	ev := libhoney.NewEvent()
	ev.AddField("name", name)
	ev.AddField("trace.trace_id", sp.traceID)
	ev.AddField("trace.parent_id", sp.spanID)
	newSpan, _ := uuid.NewRandom()
	ev.AddField("trace.span_id", newSpan.String())
	return ev
}

// PushSpanOnStack adds a new span to a trace (or creates the trace if none
// exists).
func PushSpanOnStack(ctx context.Context, name string) context.Context {
	trace := GetTraceFromContext(ctx)
	if trace == nil {
		// if we don't have an existing trace, make one and return
		trace = MakeNewTrace("", "", name)
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
	span.ev.AddField("name", name)
	newSpanList := append(trace.openSpans, span)
	trace.openSpans = newSpanList
	return ctx
}

// PushEventOnStack lets you take an event you've created outside the beeline
// and push it in to the trace. This function will assign a parent, span, and
// trace ID to the event and slot it in to the trace. A trace must exist in the
// context or this function will fail (and return an error).
func PushEventOnStack(ctx context.Context, ev *libhoney.Event) error {
	trace := GetTraceFromContext(ctx)
	if trace == nil {
		return errors.New("can't push an event on the stack without a trace in the context")
	}
	if trace.shouldDropSample {
		return nil
	}
	currentSpan := trace.openSpans[len(trace.openSpans)-1]
	// make a new span using the parent's span ID as my parent ID
	spanID, _ := uuid.NewRandom()
	span := &Span{
		timer:    timer.Start(),
		traceID:  currentSpan.traceID,
		parentID: currentSpan.spanID,
		spanID:   spanID.String(),
		ev:       ev,
	}
	newSpanList := append(trace.openSpans, span)
	trace.openSpans = newSpanList
	return nil
}

func MakeNewTrace(traceID, parentID, name string) *Trace {
	if traceID == "" {
		tid, _ := uuid.NewRandom()
		traceID = tid.String()
	}
	sid, _ := uuid.NewRandom()
	spanID := sid.String()
	ev := libhoney.NewEvent()
	span := &Span{
		timer:    timer.Start(),
		traceID:  traceID,
		spanID:   spanID,
		parentID: parentID,
		ev:       ev,
	}
	span.ev.AddField("name", name)
	span.ev.AddField("meta.root_span", true)
	// if a deterministic sampler is defined, use it. Otherwise sampling happens
	// via the hook at send time.
	var shouldDropSample bool
	var sampleRate = 1
	if sample.GlobalSampler != nil {
		shouldDropSample = !sample.GlobalSampler.Sample(traceID)
		if shouldDropSample {
			// if we're not going to send this sample, don't initialize anything.
			// We'll drop everything as it comes in to save computation, storage
			return &Trace{
				shouldDropSample: shouldDropSample,
			}
		}
		sampleRate = sample.GlobalSampler.GetSampleRate()
	}
	return &Trace{
		headers: TraceHeader{
			TraceID: traceID,
		},
		shouldDropSample: shouldDropSample,
		sampleRate:       sampleRate,
		openSpans:        []*Span{span},
		traceLevelFields: make(map[string]interface{}),
		rollupFields:     make(map[string]float64),
	}
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

func SendTrace(trace *Trace) error {
	if trace == nil {
		return nil
	}
	if trace.shouldDropSample {
		return nil
	}
	// if this trace has already been sent, complain
	if trace.sent == true {
		return errors.New("shouldn't send a trace twice.")
	}
	// if there are any remaining open spans, let's close them.
	if len(trace.openSpans) != 0 {
		for _, span := range trace.openSpans {
			span.AddField("meta.closed_by_trace_send", true)
			trace.EndCurrentSpan()
		}
	}
	for _, span := range trace.closedSpans {
		span.ev.AddField("trace.span_id", span.spanID)
		if span.parentID != "" {
			span.ev.AddField("trace.parent_id", span.parentID)
		}
		span.ev.AddField("trace.trace_id", span.traceID)
		for k, v := range trace.traceLevelFields {
			span.ev.AddField(k, v)
		}
		span.ev.SampleRate = uint(trace.sampleRate)

		// run hooks
		var shouldKeep = true
		if GlobalConfig.SamplerHook != nil {
			var sampleRate int
			shouldKeep, sampleRate = GlobalConfig.SamplerHook(span.ev.Fields())
			if shouldKeep {
				span.ev.SampleRate *= uint(sampleRate)
			} else {
				// we should drop this event
				span.shouldDrop = true
			}
		}
		if GlobalConfig.PresendHook != nil {
			// munge all the fields
			GlobalConfig.PresendHook(span.ev.Fields())
		}
		if shouldKeep {
			span.ev.SendPresampled()
		}
	}
	trace.sent = true
	return nil
}
