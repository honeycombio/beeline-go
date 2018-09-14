package trace

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/honeycombio/beeline-go/propagation"
	"github.com/honeycombio/beeline-go/sample"
	"github.com/honeycombio/beeline-go/timer"
	libhoney "github.com/honeycombio/libhoney-go"
)

var GlobalConfig Config

type Config struct {
	// SamplerHook is a function to manage sampling on this trace. See the docs
	// for `beeline.Config` for a full description.
	SamplerHook func(map[string]interface{}) (bool, int)
	// PresendHook is a function to mutate spans just before they are sent to
	// Honeycomb. See the docs for `beeline.Config` for a full description.
	PresendHook func(map[string]interface{})
}

// Trace holds some trace level state and the root of the span tree that will be
// the entire in-process trace. Traces are sent to Honeycomb when the root span
// is sent. You can send a trace manually, and that will cause all
// synchronous  spans in the trace to be sent and sent. Asynchronous spans
// must still be sent on their own
type Trace struct {
	builder          *libhoney.Builder
	traceID          string
	parentID         string
	rollupFields     map[string]float64
	rollupLock       sync.Mutex
	rootSpan         *Span
	tlfLock          sync.Mutex
	traceLevelFields map[string]interface{}
}

// NewTrace creates a brand new trace. serializedHeaders is optional, and if
// included, should be the header as written by trace.SerializeHeaders(). When
// not starting from an upstream trace, pass the empty string here.
func NewTrace(ctx context.Context, serializedHeaders string) (context.Context, *Trace) {
	trace := &Trace{
		builder:          libhoney.NewBuilder(),
		rollupFields:     make(map[string]float64),
		traceLevelFields: make(map[string]interface{}),
	}
	if serializedHeaders == "" {
		trace.traceID = uuid.Must(uuid.NewRandom()).String()
	} else {
		prop, err := propagation.UnmarshalTraceContext(serializedHeaders)
		if err == nil {
			trace.traceID = prop.TraceID
			trace.parentID = prop.ParentID
			for k, v := range prop.TraceContext {
				trace.traceLevelFields[k] = v
			}
		}
	}
	rootSpan := newSpan()
	rootSpan.isRoot = true
	if trace.parentID != "" {
		rootSpan.parentID = trace.parentID
	}
	rootSpan.ev = trace.builder.NewEvent()
	rootSpan.trace = trace
	trace.rootSpan = rootSpan

	// put trace and root span in context
	ctx = PutTraceInContext(ctx, trace)
	ctx = PutSpanInContext(ctx, rootSpan)
	return ctx, trace
}

// AddField adds a field to the trace. Every span in the trace will have this
// field added to it. These fields are also passed along to downstream services.
// It is useful to add fields here that pertain to the entire trace, to aid in
// filtering spans at many different areas of the trace together.
func (t *Trace) AddField(key string, val interface{}) {
	t.tlfLock.Lock()
	defer t.tlfLock.Unlock()
	if t.traceLevelFields != nil {
		t.traceLevelFields[key] = val
	}
}

// addRollupField is here to let a span contribute a field to the trace while
// keeping the trace's locks private.
func (t *Trace) addRollupField(key string, val float64) {
	t.rollupLock.Lock()
	defer t.rollupLock.Unlock()
	if t.rollupFields != nil {
		t.rollupFields[key] += val
	}
}

// getTraceLevelFields is here to let a span retrieve trace level fields to add
// them to itself just before sending while keeping the trace's locks around
// that field private.
func (t *Trace) getTraceLevelFields() map[string]interface{} {
	t.tlfLock.Lock()
	defer t.tlfLock.Unlock()
	// return a copy of trace level fields
	retVals := make(map[string]interface{})
	for k, v := range t.traceLevelFields {
		retVals[k] = v
	}
	return retVals
}

// GetRootSpan returns the root of the in-process trace. Sending the root span
// will send the entire trace to Honeycomb. From the root span you can walk the
// entire span tree using GetChildren (and recursively calling GetChildren on
// each child).
func (t *Trace) GetRootSpan() *Span {
	return t.rootSpan
}

// Send will finish and send all the synchronous spans in the trace to Honeycomb
func (t *Trace) Send() {
	rs := t.rootSpan
	if !rs.isSent {
		rs.Send()
		// sending the span will also send all its children
	}
}

// Span represents a specific task or portion of an application. It has a time
// and duration, and is linked to parent and children.
type Span struct {
	isAsync      bool
	isSent       bool
	isRoot       bool
	children     []*Span
	ev           *libhoney.Event
	spanID       string
	parentID     string
	parent       *Span
	rollupFields map[string]float64
	rollupLock   sync.Mutex
	timer        timer.Timer
	trace        *Trace
}

// newSpan takes care of *some* of the initialization necessary to create a new
// span. IMPORTANT it is not all of the initialization! It does *not* set parent
// ID or assign the pointer to the trace that contains this span. See existing
// uses of this function to get an example of the other things necessary to
// create a well formed span.
func newSpan() *Span {
	return &Span{
		spanID:       uuid.Must(uuid.NewRandom()).String(),
		timer:        timer.Start(),
		children:     make([]*Span, 0),
		rollupFields: make(map[string]float64),
	}
}

// AddField adds a key/value pair to this span
func (s *Span) AddField(key string, val interface{}) {
	if s.ev != nil {
		s.ev.AddField(key, val)
	}
}

// AddRollupField adds a key/value pair to this span. If it is called repeatedly
// on the same span, the values will be summed together.  Additionally, this
// field will be summed across all spans and added to the trace as a total. It
// is especially useful for doing things like adding the duration spent talking
// to a specific external service - eg database time. The root span will then
// get a field that represents the total time spent talking to the database from
// all of the spans that are part of the trace.
func (s *Span) AddRollupField(key string, val float64) {
	if s.trace != nil {
		s.trace.addRollupField(key, val)
	}
	s.rollupLock.Lock()
	defer s.rollupLock.Unlock()
	if s.rollupFields != nil {
		s.rollupFields[key] += val
	}
}

// AddTraceField adds a key/value pair to this span and all others involved in
// this trace. These fields are also passed along to downstream services. This
// method is functionally identical to `Trace.AddField()`.
func (s *Span) AddTraceField(key string, val interface{}) {
	// these get added to this span when it gets sent, so don't bother adding
	// them here
	if s.trace != nil {
		s.trace.AddField(key, val)
	}
}

// Send marks a span complete. It does some accounting and then dispatches the
// span to Honeycomb. Sending a span also triggers sending all synchronous
// child spans - in other words, if any synchronous child span has not yet been
// sent, sending the parent will finish and send the children as well.
func (s *Span) Send() {
	if s.ev == nil {
		return
	}
	// finish the timer for this span
	if s.timer != nil {
		dur := s.timer.Finish()
		s.ev.AddField("duration_ms", dur)
	}
	// set trace IDs for this span
	s.ev.AddField("trace.trace_id", s.trace.traceID)
	if s.parentID != "" {
		s.ev.AddField("trace.parent_id", s.parentID)
	}
	s.ev.AddField("trace.span_id", s.spanID)
	// add rollup fields to the event
	for k, v := range s.rollupFields {
		s.ev.AddField(k, v)
	}
	for _, child := range s.children {
		if !child.IsAsync() {
			if !child.isSent {
				child.AddField("meta.sent_by_parent", true)
				child.Send()
			}
		}
	}
	// now that we're all sent, send the span and all its children.
	s.send()
	s.isSent = true
}

// IsAsync reveals whether the span is asynchronous (true) or synchronous (false).
func (s *Span) IsAsync() bool {
	return s.isAsync
}

// GetChildren returns a list of all child spans (both synchronous and
// asynchronous).
func (s *Span) GetChildren() []*Span {
	return s.children
}

// Get Parent returns this span's parent.
func (s *Span) GetParent() *Span {
	return s.parent
}

// CreateAsyncChild creates a child of the current span that is expected to
// outlive the current span (and trace). Async spans are not automatically sent
// when their parent finishes, but are otherwise identical to synchronous spans.
func (s *Span) CreateAsyncChild(ctx context.Context) (context.Context, *Span) {
	ctx, newSpan := s.CreateChild(ctx)
	newSpan.isAsync = true
	return ctx, newSpan
}

// Span creates a synchronous child of the current span. Spans must finish
// before their parents.
func (s *Span) CreateChild(ctx context.Context) (context.Context, *Span) {
	newSpan := newSpan()
	newSpan.parent = s
	newSpan.parentID = s.spanID
	newSpan.trace = s.trace
	newSpan.ev = s.trace.builder.NewEvent()
	s.children = append(s.children, newSpan)
	ctx = PutSpanInContext(ctx, newSpan)
	return ctx, newSpan
}

// SerializeHeaders returns the trace ID, current span ID as parent ID, and an
// encoded form of all trace level fields. This serialized header is intended to
// be put in an HTTP (or other protocol) header to transmit to downstream
// services so they may start a new trace that will be connected to this trace.
// The serialized form may be passed to NewTrace() in order to create a new
// trace that will be connected to this trace.
func (s *Span) SerializeHeaders() string {
	var prop = &propagation.Propagation{
		TraceID:      s.trace.traceID,
		ParentID:     s.spanID,
		TraceContext: s.trace.traceLevelFields,
	}
	return propagation.MarshalTraceContext(prop)
}

// send gets all the trace level fields and does pre-send hooks, then sends the
// span.
func (s *Span) send() {
	// don't send already sent spans
	if s.isSent {
		return
	}
	// add all the trace level fields to the event as late as possible - when
	// the trace is all getting sent
	for k, v := range s.trace.getTraceLevelFields() {
		s.AddField(k, v)
	}

	// classify span type
	var spanType string
	switch {
	case s.isRoot:
		if s.parentID == "" {
			spanType = "root"
		} else {
			spanType = "subroot"
		}
	case s.isAsync:
		spanType = "async"
	case len(s.children) == 0:
		spanType = "leaf"
	default:
		spanType = "mid"
	}
	s.AddField("meta.span_type", spanType)

	// run hooks
	var shouldKeep = true
	if GlobalConfig.SamplerHook != nil {
		var sampleRate int
		shouldKeep, sampleRate = GlobalConfig.SamplerHook(s.ev.Fields())
		s.ev.SampleRate *= uint(sampleRate)
	} else {
		// use the default sampler
		if sample.GlobalSampler != nil {
			shouldKeep = sample.GlobalSampler.Sample(s.trace.traceID)
			s.ev.SampleRate = uint(sample.GlobalSampler.GetSampleRate())
		}
	}
	if shouldKeep {
		if GlobalConfig.PresendHook != nil {
			// munge all the fields
			GlobalConfig.PresendHook(s.ev.Fields())
		}
		s.ev.SendPresampled()
	}
}
