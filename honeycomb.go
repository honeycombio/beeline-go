package honeycomb

import (
	"context"
	"os"
	"time"

	libhoney "github.com/honeycombio/libhoney-go"
)

func NewHoneycombInstrumenter(writekey string, dataset string) {
	if dataset == "" {
		dataset = "go-http"
	}
	config := libhoney.Config{
		WriteKey: writekey,
		Dataset:  dataset,
		Output:   &libhoney.WriterOutput{},
	}
	libhoney.Init(config)

	if hostname, err := os.Hostname(); err == nil {
		libhoney.AddField("host", hostname)
	}
	return
}

// AddField allows you to add a single field to an event anywhere downstream of
// an instrumented request. After adding the appropriate middleware or wrapping
// a Handler, feel free to call AddField freely within your code. Pass it the
// context from the request (`r.Context()`) and the key and value you wish to
// add.
func AddField(ctx context.Context, key string, val interface{}) {
	ev := existingEventFromContext(ctx)
	if ev == nil {
		return
	}
	ev.AddField(key, val)
}

type Timer struct {
	start time.Time
	name  string
	ev    *libhoney.Event
}

// NewTimer is intended to be used one of two ways. To time an entire function, put
// this as the first line of the function call: `defer honeycomb.NewTimer(ctx,
// "foo", time.Now()).Finish()` To time a portion of code, save the return value
// from `honeycomb.Timer(ctx, "foo", time.Now())` and then call `.Finish()` on
// it when the timer should be stopped. For example,
// hnyTimer := honeycomb.NewTimer(ctx, "codeFragment", time.Now())
// <do stuff>
// hnyTimer.Finish()
// In both cases, the timer will be created using the name (second field) and
// have `_dur_ms` appended to the field name.
func NewTimer(ctx context.Context, name string, t time.Time) *Timer {
	ev := existingEventFromContext(ctx)
	timer := &Timer{
		start: t,
		name:  name,
		ev:    ev,
	}
	return timer
}

// Finish closes off a started timer and adds the duration to the Honeycomb event.
func (t *Timer) Finish() {
	ev := t.ev
	if ev != nil {
		dur := float64(time.Since(t.start)) / float64(time.Millisecond)
		t.ev.AddField(t.name+"_dur_ms", dur)
	}
}
