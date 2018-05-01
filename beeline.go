package beeline

import (
	"context"
	"fmt"
	"os"
	"time"

	libhoney "github.com/honeycombio/libhoney-go"
)

const (
	defaultWriteKey   = "writekey-placeholder"
	defaultDataset    = "go-http"
	defaultSampleRate = 1

	honeyBuilderContextKey = "honeycombBuilderContextKey"
	honeyEventContextKey   = "honeycombEventContextKey"
)

// Config is the place where you configure your Honeycomb write key and dataset
// name. WriteKey is the only required field in order to acutally send events to
// Honeycomb.
type Config struct {
	// Writekey is your Honeycomb authentication token, available from
	// https://ui.honeycomb.io/account. default: writekey-placeholder
	WriteKey string
	// Dataset is the name of the Honeycomb dataset to which events will be
	// sent. default: go-http
	Dataset string
	// SamplRate is a positive integer indicating the rate at which to sample
	// events. default: 1
	SampleRate uint
	// APIHost is the hostname for the Honeycomb API server to which to send
	// this event. default: https://api.honeycomb.io/
	APIHost string
	// STDOUT when set to true will print events to STDOUT *instead* of sending
	// them to honeycomb; useful for development. default: false
	STDOUT bool
	// Mute when set to true will disable Honeycomb entirely; useful for tests
	// and CI. default: false
	Mute bool
	// DisableTracing when set to true will suppress emitting trace.* fields
	DisableTracing bool
}

// Init intializes the honeycomb instrumentation library.
func Init(config Config) {
	if config.WriteKey == "" {
		config.WriteKey = defaultWriteKey
	}
	if config.Dataset == "" {
		config.Dataset = defaultDataset
	}
	if config.SampleRate == 0 {
		config.SampleRate = 1
	}
	var output libhoney.Output
	if config.STDOUT == true {
		output = &libhoney.WriterOutput{}
	}
	if config.Mute == true {
		output = &libhoney.DiscardOutput{}
	}
	libhconfig := libhoney.Config{
		WriteKey:   config.WriteKey,
		Dataset:    config.Dataset,
		SampleRate: config.SampleRate,
		Output:     output,
	}
	if config.APIHost != "" {
		libhconfig.APIHost = config.APIHost
	}
	libhoney.Init(libhconfig)
	libhoney.UserAgentAddition = fmt.Sprintf("beeline/%s", version)

	if hostname, err := os.Hostname(); err == nil {
		libhoney.AddField("meta.localhostname", hostname)
	}
	return
}

// AddField allows you to add a single field to an event anywhere downstream of
// an instrumented request. After adding the appropriate middleware or wrapping
// a Handler, feel free to call AddField freely within your code. Pass it the
// context from the request (`r.Context()`) and the key and value you wish to
// add.
func AddField(ctx context.Context, key string, val interface{}) {
	ev := ContextEvent(ctx)
	if ev == nil {
		return
	}
	ev.AddField(key, val)
}

// ContextWithEvent returns a new context created from the passed context with a
// Honeycomb event added to it. In most cases, the code adding the event to the
// context should also be responsible for sending that event on to Honeycomb
// when it's finished.
func ContextWithEvent(ctx context.Context, ev *libhoney.Event) context.Context {
	return context.WithValue(ctx, honeyEventContextKey, ev)
}

// ContextEvent retrieves the Honeycomb event from a context. You can add fields
// to the event or override settings (eg sample rate) but should not Send() the
// event; the wrapper that inserted the event into the Context is responsible
// for sending it to Hnoeycomb
func ContextEvent(ctx context.Context) *libhoney.Event {
	if ctx != nil {
		if evt, ok := ctx.Value(honeyEventContextKey).(*libhoney.Event); ok {
			return evt
		}
	}
	return nil
}

// contextBuilder isn't used yet but matches ContextEvent. When it's useful,
// export it, but until then it's just confusing.
func contextBuilder(ctx context.Context) *libhoney.Builder {
	if bldr, ok := ctx.Value(honeyBuilderContextKey).(*libhoney.Builder); ok {
		return bldr
	}
	return nil
}

// TODO move Timer to its own package; no reason it needs to be in this one.

// Timer is a convenience object to make recording how long a section of code
// takes to run a little cleaner.
type Timer struct {
	start time.Time
	name  string
	ev    *libhoney.Event
}

// NewNamedTimerC is intended to be used one of two ways. To time an entire
// function, put this as the first line of the function call:
//
// defer beeline.NewNamedTimerC(ctx, "foo", time.Now()).Finish()`
//
// To time a portion of code, save the return value from creating the timer and
// then call `.Finish()` on it when the timer should be stopped. For example,
//
// hnyTimer := beeline.NewNamedTimerC(ctx, "codeFragment", time.Now())
// <do stuff>
// hnyTimer.Finish()
//
// In both cases, the timer will be created using the name (second field) and
// have `_dur_ms` appended to the field name.
func NewNamedTimerC(ctx context.Context, name string, t time.Time) *Timer {
	ev := ContextEvent(ctx)
	return &Timer{
		start: t,
		name:  name,
		ev:    ev,
	}
}

// NewTimer will not add the results of the timing to an event from the context,
// but at least will start a timer you can use. The time passed in is used as
// the starting time, for when the thing you're timing may not be anchored on
// the current time.
func NewTimer(t time.Time) *Timer {
	return &Timer{
		start: t,
	}
}

// StartTimer records the current time for use in code.
func StartTimer() *Timer {
	return &Timer{
		start: time.Now(),
	}
}

// Finish closes off a started timer and adds the duration to the Honeycomb
// event if one is available in the stored context. Also returns the duration
// timed in milliseconds, for use when an event is not available.
func (t *Timer) Finish() float64 {
	dur := float64(time.Since(t.start)) / float64(time.Millisecond)
	if t.ev != nil {
		if t.name != "" {
			t.ev.AddField(t.name+"_dur_ms", dur)
		}
	}
	return dur
}
