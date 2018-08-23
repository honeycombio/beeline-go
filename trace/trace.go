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
	// TODO describe what these are and the return values
	SamplerHook func(map[string]interface{}) (bool, int)
	PresendHook func(map[string]interface{})
}

type Trace struct {
	// TODO add shouldDrop
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
	rootSpan.amRoot = true
	rootSpan.ev = trace.builder.NewEvent()
	rootSpan.trace = trace
	trace.rootSpan = rootSpan

	// put trace and root span in context
	ctx = PutTraceInContext(ctx, trace)
	ctx = PutSpanInContext(ctx, rootSpan)
	return ctx, trace
}

func (t *Trace) AddField(key string, val interface{}) {
	t.tlfLock.Lock()
	defer t.tlfLock.Unlock()
	if t.traceLevelFields != nil {
		t.traceLevelFields[key] = val
	}
}

func (t *Trace) GetRootSpan() *Span {
	return t.rootSpan
}

func (t *Trace) Send() {
	// TODO add sampling
	// make sure all sync spans are finished
	rs := t.rootSpan
	if !rs.amFinished {
		rs.Finish()
	}
	// start at the root span and send them all
	recursiveSend(rs)
}

// Span is the default span type.
type Span struct {
	amAsync      bool
	amFinished   bool
	amRoot       bool
	children     []*Span
	ev           *libhoney.Event
	spanID       string
	parentID     string
	parent       *Span
	rollupFields map[string]float64
	rollupLock   sync.Mutex
	sent         bool // records whether this span has already been sent.
	timer        timer.Timer
	trace        *Trace
}

// TODO don't do this - initialize all of them each place you need to; this just sets traps.
// newSpan conveniently initializes some (but not all) things that would
// otherwise be nil
func newSpan() *Span {
	return &Span{
		spanID:       uuid.Must(uuid.NewRandom()).String(),
		timer:        timer.Start(),
		children:     make([]*Span, 0),
		rollupFields: make(map[string]float64),
	}
}

func (s *Span) AddField(key string, val interface{}) {
	if s.ev != nil {
		s.ev.AddField(key, val)
	}
}

func (s *Span) AddRollupField(key string, val float64) {
	s.rollupLock.Lock()
	if s.rollupFields != nil {
		s.rollupFields[key] += val
	}
	s.rollupLock.Unlock()
	if s.trace != nil {
		s.trace.rollupLock.Lock()
		defer s.trace.rollupLock.Unlock()
		s.trace.rollupFields[key] += val
	}
}

func (s *Span) AddTraceField(key string, val interface{}) {
	// these get added to this span when it gets sent, so don't bother adding
	// them here
	if s.trace != nil {
		s.trace.AddField(key, val)
	}
}

func (s *Span) Finish() {
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
		if !child.AmAsync() {
			if !child.amFinished {
				child.AddField("meta.finished_by_parent", true)
				child.Finish()
			}
		}
	}
	s.amFinished = true
	// if we're closing the root span, send the whole trace
	if s.amRoot {
		s.trace.Send()
	}
}

func (s *Span) AmAsync() bool {
	return s.amAsync
}

func (s *Span) GetParent() *Span {
	return s.parent
}

// ChildAsyncSpan creates a child of the current span that is expected to
// outlive the current span (and trace). Async spans must be manually sent using
// the `Send()` method but are otherwise identical to normal spans.
func (s *Span) ChildAsyncSpan(ctx context.Context) (context.Context, *Span) {
	ctx, newSpan := s.ChildSpan(ctx)
	newSpan.amAsync = true
	return ctx, newSpan
}

// Span creates a child of the current span. Spans must finish before their
// parents.
func (s *Span) ChildSpan(ctx context.Context) (context.Context, *Span) {
	newSpan := newSpan()
	newSpan.parent = s
	newSpan.parentID = s.spanID
	newSpan.trace = s.trace
	newSpan.ev = s.trace.builder.NewEvent()
	s.children = append(s.children, newSpan)
	ctx = PutSpanInContext(ctx, newSpan)
	return ctx, newSpan
}

func (s *Span) SerializeHeaders() string {
	var prop = &propagation.Propagation{
		TraceID:      s.trace.traceID,
		ParentID:     s.spanID,
		TraceContext: s.trace.traceLevelFields,
	}
	// prop.Source = HeaderSourceBeeline

	return propagation.MarshalTraceContext(prop)
}

// send gets all the trace level fields and does pre-send hooks, then sends the
// span.
func (s *Span) send() {
	// don't send already sent spans
	if s.sent {
		return
	}
	// add all the trace level fields to the event as late as possible - when
	// the trace is all getting sent
	s.trace.tlfLock.Lock()
	for k, v := range s.trace.traceLevelFields {
		s.AddField(k, v)
	}
	s.trace.tlfLock.Unlock()

	// classify span type
	var spanType string
	switch {
	case s.amRoot:
		if s.parentID == "" {
			spanType = "root"
		} else {
			spanType = "subroot"
		}
	case s.amAsync:
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

// recursiveSend sends this span and then all its children; async spans don't
// get sent here
func recursiveSend(s *Span) {
	if !s.sent {
		s.send()
	}
	for _, childSpan := range s.children {
		if !childSpan.AmAsync() {
			recursiveSend(childSpan)
		}
	}
	s.sent = true
}

// Send sends this span and any synchronous span children. Does not send any
// async children. Primarily used on async spans. While you can call `Send` on a
// synchronous (normal) span, doing so prevents the span from getting any trace
// level fields added after it is sent. Synchronous spans get automatically sent
// when the trace finishes; it should never be necessary to call `Send` on a
// synchronous span. (but if you do, it will in fact get sent.)
func (s *Span) Send() {
	if !s.amFinished {
		s.Finish()
	}
	recursiveSend(s)
}
