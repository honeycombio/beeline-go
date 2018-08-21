package internal

// import (
// 	"context"
// 	"errors"
// 	"sync"

// 	"github.com/google/uuid"

// 	"github.com/honeycombio/beeline-go/internal/sample"
// 	"github.com/honeycombio/beeline-go/timer"
// 	libhoney "github.com/honeycombio/libhoney-go"
// )

// // a short discussion about the model of managing traces.
// //
// // The user of the beeline only cares about traces when wanting to assert the
// // trace ID upon its creation. In most cases, the user is only interacting with
// // spans.
// //
// // Everything about interacting with a trace or its spans comes from providing
// // the context that's storing the trace and the current span and acting upon
// // that information.  It's the responsibility of the user to capture the context
// // coming back from each span creation in order to add fields to the right span
// //
// // The current span has a link back to the trace for adding trace level fields
// // and to trigger sending all spans when the root span is finished.
// //
// // Whet an async span is started, it's still part of the trace, but does not get
// // sent when the root span finishes. These spans are intended to outlive their
// // parent process.
// //
// // Spans (even async spans) are meant to be started and finished by the same
// // process. They are not intended to be started in one service, serialized and
// // passed to a second service, and finished there.
// //
// // You can serialize trace context and pass that along to downstream services,
// // but that downstream service should create a trace of its own (that shares the
// // trace ID so it will be unified in the Honeycomb UI) with its own spans.

var GlobalConfig Config

type Config struct {
	SamplerHook func(map[string]interface{}) (bool, int)
	PresendHook func(map[string]interface{})
}

// Trace holds fields relevant to the entire trace - a list of spans, a list of
// trace level fields, and so on.  The trace does not know what the "current"
// span is, since there can be multiple current spans when dealing with
// goroutines.
type Trace struct {
	// shouldDrop is true when this trace should be dropped, false when it
	// should be sent.
	shouldDrop bool
	sampleRate int

	headers          TraceHeader
	sent             bool // true when the trace is sent, false otherwise.
	spans            []*Span
	spanLock         sync.Mutex
	rollupFields     map[string]float64
	rollupLock       sync.Mutex
	traceLevelFields map[string]interface{}
	tlfLock          sync.Mutex
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
	// shouldDrop is used by sampler hook to pass information that this specific
	// span should be dropped instead of sent
	shouldDrop bool
	// timer starts when the span is created and ends when it is closed to get a
	// duration for this span's existence.
	timer timer.Timer
	// amRoot is true when this span is the root span for the trace.  When the
	// root span is closed, the rest of the trace should be wrapped up and sent
	// as well.
	amRoot bool
	// amAsync is true when this span is an async span.  Async spans get sent
	// immediately when they finish, which is usually after the rest of the span
	// has already finished. They are intended to outlive the trace, and are
	// useful for things like background email sending that usually takes longer
	// than the main trace wants to wait for. trace is a pointer to the trace
	// that contains this span, so that when ending a span you can get back to
	// the trace to move it around in the internal trace span tree accounting
	// data structures appropriately.
	amAsync bool
	// trace is a pointer to the trace that contains this span, so that when
	// ending a span you can get back to the trace to move it around in the
	// internal trace span tree accounting data structures appropriately.
	trace *Trace
	// parent is a pointer to the span that spawned this span, so when this span
	// finishes, we can put the parent back in the context.
	parent *Span
	// hasFinished is set to true when the span is closed or finished. This does
	// not trigger the span to get sent to Honeycomb - that happens when the
	// entire trace is closed. Whether a span has finished is tracked to help
	// identify unfinished spans as potential bugs in the surrounding span
	// management, and indicate when maybe an async span should be created
	// instead
	hasFinished bool

	// three IDs to identify the span
	traceID  string
	spanID   string
	parentID string

	// ev has all the fields added to this span ready to be sent to Honeycomb
	// when the time is right
	ev *libhoney.Event
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

// AddField on the trace object adds the key/val provided to every span in the
// trace
func (t *Trace) AddField(key string, val interface{}) {
	if t.shouldDrop {
		return
	}
	t.tlfLock.Lock()
	defer t.tlfLock.Unlock()
	if t.traceLevelFields != nil {
		t.traceLevelFields[key] = val
	}
}

// AddSpan adds a new span to the trace to be tracked
func (t *Trace) AddSpan(span *Span) {
	t.spanLock.Lock()
	defer t.spanLock.Unlock()
	t.spans = append(t.spans, span)
}

func (t *Trace) Send() error {
	// if we're not supposed to send this trace because of sampling, don't.
	if t.shouldDrop {
		return nil
	}
	// if this trace has already been sent, complain
	if t.sent == true {
		return errors.New("shouldn't send a trace twice.")
	}
	// go through all the spans and send them!
	for _, span := range t.spans {
		// skip async spans when sending the trace; they are supposed to outlive
		// the trace.
		if span.amAsync {
			continue
		}
		// Everything else should get marked if it is getting closed by the
		// trace send.
		if !span.hasFinished {
			span.AddField("meta.closed_by_trace_send", true)
		}

		// spew.Dump(span)
		span.Send()

	}
	t.sent = true
	return nil
}

func (s *Span) AddField(key string, val interface{}) {
	s.ev.AddField(key, val)
}

// AddRollupField adds the key and value to the current span and also adds the
// sum of all times this is called to the root span of the trace
func (s *Span) AddRollupField(key string, val float64) {
	if s.shouldDrop {
		return
	}
	s.ev.AddField(key, val)
	s.trace.rollupLock.Lock()
	defer s.trace.rollupLock.Unlock()
	s.trace.rollupFields[key] += val
}

func (s *Span) Finish(ctx context.Context) context.Context {
	if s.shouldDrop {
		// we're not recording this trace; we're done here.
		if s.parent != nil {
			ctx = PutCurrentSpanInContext(ctx, s.parent)
		}
		return ctx
	}
	s.hasFinished = true

	// finish the timer and add duration to the span
	dur := s.timer.Finish()
	s.AddField("duration_ms", dur)

	// if we're an async span, send immediately
	if s.amAsync {
		s.Send()
	}
	// if we're finishing the root span, we should send the whole trace.
	if s.amRoot {
		s.trace.rollupLock.Lock()
		for k, v := range s.trace.rollupFields {
			s.AddField(k, v)
		}
		s.trace.rollupLock.Unlock()
		s.trace.Send()
	}
	// if we have a parent span, we should set that as the new current.
	if s.parent != nil {
		ctx = PutCurrentSpanInContext(ctx, s.parent)
	}
	return ctx
}

// Send goes through all the accounting necessary and then actually dispatches
// this span's event to Honeycomb
func (s *Span) Send() {
	// add all the relevant IDs
	s.ev.AddField("trace.span_id", s.spanID)
	if s.parentID != "" {
		s.ev.AddField("trace.parent_id", s.parentID)
	}
	s.ev.AddField("trace.trace_id", s.traceID)

	s.trace.tlfLock.Lock()
	for k, v := range s.trace.traceLevelFields {
		s.AddField(k, v)
	}
	s.trace.tlfLock.Unlock()

	// run hooks
	var shouldKeep = true
	if GlobalConfig.SamplerHook != nil {
		var sampleRate int
		shouldKeep, sampleRate = GlobalConfig.SamplerHook(s.ev.Fields())
		s.ev.SampleRate *= uint(sampleRate)
	}
	if GlobalConfig.PresendHook != nil {
		// munge all the fields
		GlobalConfig.PresendHook(s.ev.Fields())
	}
	if shouldKeep {
		s.ev.SendPresampled()
	}
}

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

// StartSpan adds a new span to a trace (or creates the trace if none
// exists).
func StartSpan(ctx context.Context, name string) (context.Context, *Span) {
	trace := GetTraceFromContext(ctx)
	if trace == nil {
		// if we don't have an existing trace, make one and return
		span := MakeNewTrace("", "", name)
		ctx = PutCurrentSpanInContext(ctx, span)
		ctx = PutTraceInContext(ctx, span.trace)
		return ctx, span
	}
	currentSpan := CurrentSpan(ctx)
	// make a new span using the parent's span ID as my parent ID
	spanID, _ := uuid.NewRandom()
	span := &Span{
		timer:    timer.Start(),
		trace:    trace,
		parent:   currentSpan,
		traceID:  currentSpan.traceID,
		parentID: currentSpan.spanID,
		spanID:   spanID.String(),
		ev:       libhoney.NewEvent(),
	}
	span.ev.SampleRate = uint(trace.sampleRate)
	span.ev.AddField("name", name)
	trace.AddSpan(span)
	ctx = PutCurrentSpanInContext(ctx, span)
	return ctx, span
}

// StartSpanWithEvent lets you take an event you've created outside the beeline
// and push it in to the trace. This function will assign a parent, span, and
// trace ID to the event and slot it in to the trace.
func StartSpanWithEvent(ctx context.Context, ev *libhoney.Event) (context.Context, *Span) {
	var span *Span
	ctx, span = StartSpan(ctx, "")
	span.ev = ev
	return ctx, span
}

func StartAsyncSpan(ctx context.Context, name string) (context.Context, *Span) {
	var span *Span
	ctx, span = StartSpan(ctx, "")
	span.amAsync = true
	return ctx, span
}

func StartTraceWithIDs(ctx context.Context, traceID, parentID, name string) (context.Context, *Span) {
	span := MakeNewTrace(traceID, parentID, name)
	ctx = PutCurrentSpanInContext(ctx, span)
	ctx = PutTraceInContext(ctx, span.trace)
	return ctx, span
}

func MakeNewTrace(traceID, parentID, name string) *Span {
	// TODO start up something to catch if the context gets canceled or times
	// out and sends the trace if so -- is this reasonable? maybe a config
	// option on the trace itself?
	if traceID == "" {
		tid, _ := uuid.NewRandom()
		traceID = tid.String()
	}
	sid, _ := uuid.NewRandom()
	spanID := sid.String()

	trace := &Trace{
		headers: TraceHeader{
			TraceID: traceID,
		},
		spans:            make([]*Span, 0, 2), // most traces will have at least 2 spans
		traceLevelFields: make(map[string]interface{}),
		rollupFields:     make(map[string]float64),
	}
	// if a deterministic sampler is defined, use it. Otherwise sampling happens
	// via the hook at send time.
	var shouldDrop bool
	var sampleRate = 1
	if sample.GlobalSampler != nil {
		shouldDrop = !sample.GlobalSampler.Sample(traceID)
		sampleRate = sample.GlobalSampler.GetSampleRate()
	}
	trace.shouldDrop = shouldDrop
	trace.sampleRate = sampleRate

	span := &Span{
		shouldDrop: shouldDrop,
		timer:      timer.Start(),
		amRoot:     true,
		trace:      trace,
		traceID:    traceID,
		spanID:     spanID,
		parentID:   parentID,
		ev:         libhoney.NewEvent(),
	}
	span.ev.SampleRate = uint(trace.sampleRate)
	span.ev.AddField("name", name)
	span.ev.AddField("meta.root_span", true)

	// add the newly formed span to the trace and add both to the context
	trace.AddSpan(span)
	return span
}

// FinishSpan "closes" the current span by popping it off the open stack and
// putting it on the closed stack. It is not sent in case additional trace level
// fields get added they will still make it onto the closed spans. The returned
// context has the parent of this span put back in place as "current".
func FinishSpan(ctx context.Context) context.Context {
	span := GetCurrentSpanFromContext(ctx)
	if span == nil {
		// we've somehow lost context.
		// TODO This is an error we should flag somehow
		return ctx
	}
	return span.Finish(ctx)
}

// CurrentSpan gets the span marked current in the context. Returns nil when
// there are no spans.
func CurrentSpan(ctx context.Context) *Span {
	return GetCurrentSpanFromContext(ctx)
}
// // Trace holds fields relevant to the entire trace - a list of spans, a list of
// // trace level fields, and so on.  The trace does not know what the "current"
// // span is, since there can be multiple current spans when dealing with
// // goroutines.
// type Trace struct {
// 	// shouldDrop is true when this trace should be dropped, false when it
// 	// should be sent.
// 	shouldDrop bool
// 	sampleRate int

// 	headers          TraceHeader
// 	sent             bool // true when the trace is sent, false otherwise.
// 	spans            []*Span
// 	spanLock         sync.Mutex
// 	rollupFields     map[string]float64
// 	rollupLock       sync.Mutex
// 	traceLevelFields map[string]interface{}
// 	tlfLock          sync.Mutex
// }

// // TODO give spans a pointer back to the trace they're in so that you can end a
// // specific span rather than only the current one. Necessary to allow concurrent
// // use of a trace.
// type Span struct {
// 	// shouldDrop is used by sampler hook to pass information that this specific
// 	// span should be dropped instead of sent
// 	shouldDrop bool
// 	// timer starts when the span is created and ends when it is closed to get a
// 	// duration for this span's existence.
// 	timer timer.Timer
// 	// amRoot is true when this span is the root span for the trace.  When the
// 	// root span is closed, the rest of the trace should be wrapped up and sent
// 	// as well.
// 	amRoot bool
// 	// amAsync is true when this span is an async span.  Async spans get sent
// 	// immediately when they finish, which is usually after the rest of the span
// 	// has already finished. They are intended to outlive the trace, and are
// 	// useful for things like background email sending that usually takes longer
// 	// than the main trace wants to wait for. trace is a pointer to the trace
// 	// that contains this span, so that when ending a span you can get back to
// 	// the trace to move it around in the internal trace span tree accounting
// 	// data structures appropriately.
// 	amAsync bool
// 	// trace is a pointer to the trace that contains this span, so that when
// 	// ending a span you can get back to the trace to move it around in the
// 	// internal trace span tree accounting data structures appropriately.
// 	trace *Trace
// 	// parent is a pointer to the span that spawned this span, so when this span
// 	// finishes, we can put the parent back in the context.
// 	parent *Span
// 	// hasFinished is set to true when the span is closed or finished. This does
// 	// not trigger the span to get sent to Honeycomb - that happens when the
// 	// entire trace is closed. Whether a span has finished is tracked to help
// 	// identify unfinished spans as potential bugs in the surrounding span
// 	// management, and indicate when maybe an async span should be created
// 	// instead
// 	hasFinished bool

// 	// three IDs to identify the span
// 	traceID  string
// 	spanID   string
// 	parentID string

// 	// ev has all the fields added to this span ready to be sent to Honeycomb
// 	// when the time is right
// 	ev *libhoney.Event
// }

// type HeaderSource int

// const (
// 	HeaderSourceUnknown HeaderSource = iota
// 	HeaderSourceBeeline
// 	HeaderSourceAmazon
// 	HeaderSourceZipkin
// 	HeaderSourceJaeger
// )

// type TraceHeader struct {
// 	Source   HeaderSource
// 	TraceID  string
// 	ParentID string
// 	SpanID   string
// }

// // AddField on the trace object adds the key/val provided to every span in the
// // trace
// func (t *Trace) AddField(key string, val interface{}) {
// 	if t.shouldDrop {
// 		return
// 	}
// 	t.tlfLock.Lock()
// 	defer t.tlfLock.Unlock()
// 	if t.traceLevelFields != nil {
// 		t.traceLevelFields[key] = val
// 	}
// }

// // AddSpan adds a new span to the trace to be tracked
// func (t *Trace) AddSpan(span *Span) {
// 	t.spanLock.Lock()
// 	defer t.spanLock.Unlock()
// 	t.spans = append(t.spans, span)
// }

// func (t *Trace) Send() error {
// 	// if we're not supposed to send this trace because of sampling, don't.
// 	if t.shouldDrop {
// 		return nil
// 	}
// 	// if this trace has already been sent, complain
// 	if t.sent == true {
// 		return errors.New("shouldn't send a trace twice.")
// 	}
// 	// go through all the spans and send them!
// 	for _, span := range t.spans {
// 		// skip async spans when sending the trace; they are supposed to outlive
// 		// the trace.
// 		if span.amAsync {
// 			continue
// 		}
// 		// Everything else should get marked if it is getting closed by the
// 		// trace send.
// 		if !span.hasFinished {
// 			span.AddField("meta.closed_by_trace_send", true)
// 		}

// 		// spew.Dump(span)
// 		span.Send()

// 	}
// 	t.sent = true
// 	return nil
// }

// func (s *Span) AddField(key string, val interface{}) {
// 	s.ev.AddField(key, val)
// }

// // AddRollupField adds the key and value to the current span and also adds the
// // sum of all times this is called to the root span of the trace
// func (s *Span) AddRollupField(key string, val float64) {
// 	if s.shouldDrop {
// 		return
// 	}
// 	s.ev.AddField(key, val)
// 	s.trace.rollupLock.Lock()
// 	defer s.trace.rollupLock.Unlock()
// 	s.trace.rollupFields[key] += val
// }

// func (s *Span) Finish(ctx context.Context) context.Context {
// 	if s.shouldDrop {
// 		// we're not recording this trace; we're done here.
// 		if s.parent != nil {
// 			ctx = PutCurrentSpanInContext(ctx, s.parent)
// 		}
// 		return ctx
// 	}
// 	s.hasFinished = true

// 	// finish the timer and add duration to the span
// 	dur := s.timer.Finish()
// 	s.AddField("duration_ms", dur)

// 	// if we're an async span, send immediately
// 	if s.amAsync {
// 		s.Send()
// 	}
// 	// if we're finishing the root span, we should send the whole trace.
// 	if s.amRoot {
// 		s.trace.rollupLock.Lock()
// 		for k, v := range s.trace.rollupFields {
// 			s.AddField(k, v)
// 		}
// 		s.trace.rollupLock.Unlock()
// 		s.trace.Send()
// 	}
// 	// if we have a parent span, we should set that as the new current.
// 	if s.parent != nil {
// 		ctx = PutCurrentSpanInContext(ctx, s.parent)
// 	}
// 	return ctx
// }

// // Send goes through all the accounting necessary and then actually dispatches
// // this span's event to Honeycomb
// func (s *Span) Send() {
// 	// add all the relevant IDs
// 	s.ev.AddField("trace.span_id", s.spanID)
// 	if s.parentID != "" {
// 		s.ev.AddField("trace.parent_id", s.parentID)
// 	}
// 	s.ev.AddField("trace.trace_id", s.traceID)

// 	s.trace.tlfLock.Lock()
// 	for k, v := range s.trace.traceLevelFields {
// 		s.AddField(k, v)
// 	}
// 	s.trace.tlfLock.Unlock()

// 	// run hooks
// 	var shouldKeep = true
// 	if GlobalConfig.SamplerHook != nil {
// 		var sampleRate int
// 		shouldKeep, sampleRate = GlobalConfig.SamplerHook(s.ev.Fields())
// 		s.ev.SampleRate *= uint(sampleRate)
// 	}
// 	if GlobalConfig.PresendHook != nil {
// 		// munge all the fields
// 		GlobalConfig.PresendHook(s.ev.Fields())
// 	}
// 	if shouldKeep {
// 		s.ev.SendPresampled()
// 	}
// }

// // AddField gets the current span and adds the field as is - it does not give
// // the field a prefix in the way the public beeline API does. This is necessary
// // to add protected fields such as `name` or `duration_ms`
// func AddField(ctx context.Context, key string, val interface{}) {
// 	span := CurrentSpan(ctx)
// 	if span != nil {
// 		if span.ev != nil {
// 			span.ev.AddField(key, val)
// 		}
// 	}
// }

// // StartSpan adds a new span to a trace (or creates the trace if none
// // exists).
// func StartSpan(ctx context.Context, name string) (context.Context, *Span) {
// 	trace := GetTraceFromContext(ctx)
// 	if trace == nil {
// 		// if we don't have an existing trace, make one and return
// 		span := MakeNewTrace("", "", name)
// 		ctx = PutCurrentSpanInContext(ctx, span)
// 		ctx = PutTraceInContext(ctx, span.trace)
// 		return ctx, span
// 	}
// 	currentSpan := CurrentSpan(ctx)
// 	// make a new span using the parent's span ID as my parent ID
// 	spanID, _ := uuid.NewRandom()
// 	span := &Span{
// 		timer:    timer.Start(),
// 		trace:    trace,
// 		parent:   currentSpan,
// 		traceID:  currentSpan.traceID,
// 		parentID: currentSpan.spanID,
// 		spanID:   spanID.String(),
// 		ev:       libhoney.NewEvent(),
// 	}
// 	span.ev.SampleRate = uint(trace.sampleRate)
// 	span.ev.AddField("name", name)
// 	trace.AddSpan(span)
// 	ctx = PutCurrentSpanInContext(ctx, span)
// 	return ctx, span
// }

// // StartSpanWithEvent lets you take an event you've created outside the beeline
// // and push it in to the trace. This function will assign a parent, span, and
// // trace ID to the event and slot it in to the trace.
// func StartSpanWithEvent(ctx context.Context, ev *libhoney.Event) (context.Context, *Span) {
// 	var span *Span
// 	ctx, span = StartSpan(ctx, "")
// 	span.ev = ev
// 	return ctx, span
// }

// func StartAsyncSpan(ctx context.Context, name string) (context.Context, *Span) {
// 	var span *Span
// 	ctx, span = StartSpan(ctx, "")
// 	span.amAsync = true
// 	return ctx, span
// }

// func StartTraceWithIDs(ctx context.Context, traceID, parentID, name string) (context.Context, *Span) {
// 	span := MakeNewTrace(traceID, parentID, name)
// 	ctx = PutCurrentSpanInContext(ctx, span)
// 	ctx = PutTraceInContext(ctx, span.trace)
// 	return ctx, span
// }

// func MakeNewTrace(traceID, parentID, name string) *Span {
// 	// TODO start up something to catch if the context gets canceled or times
// 	// out and sends the trace if so -- is this reasonable? maybe a config
// 	// option on the trace itself?
// 	if traceID == "" {
// 		tid, _ := uuid.NewRandom()
// 		traceID = tid.String()
// 	}
// 	sid, _ := uuid.NewRandom()
// 	spanID := sid.String()

// 	trace := &Trace{
// 		headers: TraceHeader{
// 			TraceID: traceID,
// 		},
// 		spans:            make([]*Span, 0, 2), // most traces will have at least 2 spans
// 		traceLevelFields: make(map[string]interface{}),
// 		rollupFields:     make(map[string]float64),
// 	}
// 	// if a deterministic sampler is defined, use it. Otherwise sampling happens
// 	// via the hook at send time.
// 	var shouldDrop bool
// 	var sampleRate = 1
// 	if sample.GlobalSampler != nil {
// 		shouldDrop = !sample.GlobalSampler.Sample(traceID)
// 		sampleRate = sample.GlobalSampler.GetSampleRate()
// 	}
// 	trace.shouldDrop = shouldDrop
// 	trace.sampleRate = sampleRate

// 	span := &Span{
// 		shouldDrop: shouldDrop,
// 		timer:      timer.Start(),
// 		amRoot:     true,
// 		trace:      trace,
// 		traceID:    traceID,
// 		spanID:     spanID,
// 		parentID:   parentID,
// 		ev:         libhoney.NewEvent(),
// 	}
// 	span.ev.SampleRate = uint(trace.sampleRate)
// 	span.ev.AddField("name", name)
// 	span.ev.AddField("meta.root_span", true)

// 	// add the newly formed span to the trace and add both to the context
// 	trace.AddSpan(span)
// 	return span
// }

// // FinishSpan "closes" the current span by popping it off the open stack and
// // putting it on the closed stack. It is not sent in case additional trace level
// // fields get added they will still make it onto the closed spans. The returned
// // context has the parent of this span put back in place as "current".
// func FinishSpan(ctx context.Context) context.Context {
// 	span := GetCurrentSpanFromContext(ctx)
// 	if span == nil {
// 		// we've somehow lost context.
// 		// TODO This is an error we should flag somehow
// 		return ctx
// 	}
// 	return span.Finish(ctx)
// }

// // CurrentSpan gets the span marked current in the context. Returns nil when
// // there are no spans.
// func CurrentSpan(ctx context.Context) *Span {
// 	return GetCurrentSpanFromContext(ctx)
// }
