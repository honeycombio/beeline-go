package trace

type Trace struct{}

func NewTrace() *Trace {}

func NewSubTrace(traceContext string) *Trace {}

func (t *Trace) AddField() {}

func (t *Trace) GetRootSpan() *Span {}

func (t *Trace) Send() {}

func (t *Trace) SerializeTraceContext() string {}

type Span struct{}

func (s *Span) AddField() {}

func (s *Span) AddRollupField() {}

func (s *Span) AddTraceField() {}

func (s *Span) Finish() {}

func (s *Span) GetParent() *Span {}

func (s *Span) NewAsyncSpan() {}

func (s *Span) NewSpan() *Span {}

func (s *Span) send() {}

type AsyncSpan struct {
	Span
}

func (a *AsyncSpan) Send() {}
