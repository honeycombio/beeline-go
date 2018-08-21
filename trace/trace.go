package trace

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/honeycombio/beeline-go/propagation"
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
	rootSpan         Span
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
	rootSpan := newSpan().(*SyncSpan)
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

func (t *Trace) GetRootSpan() Span {
	return t.rootSpan
}

func (t *Trace) Send() {
	// TODO add sampling
	// make sure all sync spans are finished
	rs := t.rootSpan.(*SyncSpan)
	if !rs.amFinished {
		rs.Finish()
	}
	// start at the root span and send them all
	recursiveSend(rs)
}

// Span is fulfilled by both synchronous and asynchronous spans
type Span interface {
	AddField(string, interface{})
	AddRollupField(string, float64)
	AddTraceField(string, interface{})
	AmAsync() bool
	ChildAsyncSpan(context.Context) (context.Context, Span)
	ChildSpan(context.Context) (context.Context, Span)
	Finish()
	GetParent() Span
	SerializeHeaders() string
}

// SyncSpan is the default span type.
type SyncSpan struct {
	amAsync      bool
	amFinished   bool
	amRoot       bool
	children     []Span
	ev           *libhoney.Event
	spanID       string
	parentID     string
	parent       Span
	rollupFields map[string]float64
	rollupLock   sync.Mutex
	timer        timer.Timer
	trace        *Trace
}

// TODO don't do this - initialize all of them each place you need to; this just sets traps.
// newSpan conveniently initializes some (but not all) things that would
// otherwise be nil
func newSpan() Span {
	return &SyncSpan{
		spanID:       uuid.Must(uuid.NewRandom()).String(),
		timer:        timer.Start(),
		children:     make([]Span, 0),
		rollupFields: make(map[string]float64),
	}
}

func (s *SyncSpan) AddField(key string, val interface{}) {
	if s.ev != nil {
		s.ev.AddField(key, val)
	}
}

func (s *SyncSpan) AddRollupField(key string, val float64) {
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

func (s *SyncSpan) AddTraceField(key string, val interface{}) {
	// these get added to this span when it gets sent, so don't bother adding
	// them here
	if s.trace != nil {
		s.trace.AddField(key, val)
	}
}

func (s *SyncSpan) Finish() {
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
	// TODO finish all unfinished non-async children; identify they were
	// finished by the parent rather than themselves
	for _, child := range s.children {
		if !child.AmAsync() {
			childSpan := child.(*SyncSpan)
			if !childSpan.amFinished {
				childSpan.AddField("meta.finished_by_parent", true)
				childSpan.Finish()
			}
		}
	}
	s.amFinished = true
	// if we're closing the root span, send the whole trace
	if s.amRoot {
		s.trace.Send()
	}
}

func (s *SyncSpan) AmAsync() bool {
	return s.amAsync
}

func (s *SyncSpan) GetParent() Span {
	return s.parent
}

// AsyncSpan creates a child of the current span that is expected to outlive
// the current span (and trace). Async spans must be manually sent using the
// `Send()` method but are otherwise identical to normal spans.
func (s *SyncSpan) ChildAsyncSpan(ctx context.Context) (context.Context, Span) {
	ctx, syncChild := s.ChildSpan(ctx)
	newSpan := &AsyncSpan{
		SyncSpan: *(syncChild.(*SyncSpan)),
	}
	newSpan.amAsync = true
	ctx = PutSpanInContext(ctx, newSpan)
	return ctx, newSpan
}

// Span creates a child of the current span. Spans must finish before their parents.
func (s *SyncSpan) ChildSpan(ctx context.Context) (context.Context, Span) {
	newSpan := newSpan().(*SyncSpan)
	newSpan.parent = s
	newSpan.parentID = s.spanID
	newSpan.trace = s.trace
	newSpan.ev = s.trace.builder.NewEvent()
	s.children = append(s.children, newSpan)
	ctx = PutSpanInContext(ctx, newSpan)
	return ctx, newSpan
}

func (s *SyncSpan) SerializeHeaders() string {
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
func (s *SyncSpan) send() {
	// add all the trace level fields to the event as late as possible - when
	// the trace is all getting sent
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

// recursiveSend sends this span and then all its children; async spans don't
// get sent here
func recursiveSend(s *SyncSpan) {
	if !s.amAsync {
		s.send()
		for _, childSpan := range s.children {
			if !childSpan.AmAsync() {
				recursiveSend(childSpan.(*SyncSpan))
			}
		}
	}
}

// AsyncSpan does all the things a span does except get sent when the trace
// finishes. You must explicitly send AsyncSpans when they are ready
type AsyncSpan struct {
	SyncSpan
}

// Send sends this span and any synchronous span children. Any AsyncSpan
// children must still be sent manually. TODO it's likely that using an
// interface and two different types here is a terrible idea and syncness should
// just be an attribute of Span rather than a separate type. Send on a sync span
// would ... be a noop? not sure.
func (a *AsyncSpan) Send() {
	if !a.amFinished {
		a.Finish()
	}
	a.send()
	for _, childSpan := range a.children {
		if !childSpan.AmAsync() {
			recursiveSend(childSpan.(*SyncSpan))
		}
	}
}
