package beeline

import (
	"context"
	"fmt"
	"os"

	libhoney "github.com/honeycombio/libhoney-go"
)

const (
	defaultWriteKey   = "writekey-placeholder"
	defaultDataset    = "beeline-go"
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
	// Service Name identifies your application. While optional, setting this
	// field is extremely valuable when you instrument multiple services. If set
	// it will be added to all events as `service_name`
	ServiceName string
	// SampleRate is a positive integer indicating the rate at which to sample
	// events. default: 1
	SampleRate uint
	// DeterministicSample is a field name to deterministically sample on, i.e.,
	// sample 1/N based on content of this field. default: `trace.trace_id`
	DeterministicSample string
	// APIHost is the hostname for the Honeycomb API server to which to send
	// this event. default: https://api.honeycomb.io/
	APIHost string
	// STDOUT when set to true will print events to STDOUT *instead* of sending
	// them to honeycomb; useful for development. default: false
	STDOUT bool
	// Mute when set to true will disable Honeycomb entirely; useful for tests
	// and CI. default: false
	Mute bool
	// Debug will emit verbose logging to STDOUT when true. If you're having
	// trouble getting the beeline to work, set this to true in a dev
	// environment.
	Debug bool

	// disableTracing when set to true will suppress emitting trace.* fields
	// disableTracing bool
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
	if config.DeterministicSample == "" {
		// Always present in beeline
		config.DeterministicSample = "trace.trace_id"
	}
	var output libhoney.Output
	if config.STDOUT == true {
		output = &libhoney.WriterOutput{}
	}
	if config.Mute == true {
		output = &libhoney.DiscardOutput{}
	}
	libhconfig := libhoney.Config{
		WriteKey:            config.WriteKey,
		Dataset:             config.Dataset,
		SampleRate:          config.SampleRate,
		DeterministicSample: config.DeterministicSample,
		Output:              output,
	}
	if config.APIHost != "" {
		libhconfig.APIHost = config.APIHost
	}
	libhoney.Init(libhconfig)

	// set the version in both the useragent and in all events
	libhoney.UserAgentAddition = fmt.Sprintf("beeline/%s", version)
	libhoney.AddField("meta.beeline_version", version)

	// add a bunch of fields
	if config.ServiceName != "" {
		libhoney.AddField("service_name", config.ServiceName)
	}
	if hostname, err := os.Hostname(); err == nil {
		libhoney.AddField("meta.local_hostname", hostname)
	}

	if config.Debug {
		// TODO add more debugging than just the responses queue
		go readResponses(libhoney.Responses())
	}
	return
}

// Flush sends any pending events to Honeycomb. This is optional; events will be
// flushed on a timer otherwies. It is useful to flush befare AWS Lambda
// functions finish to ensure events get sent before AWS freezes the function.
func Flush() {
	libhoney.Flush()
}

// Close shuts down the beeline. Closing flushes any pending events and blocks
// until they have been sent. It is optional to close the beeline, and
// prohibited to try and send an event after the beeline has been closed.
func Close() {
	libhoney.Close()
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
	namespacedKey := fmt.Sprintf("app.%s", key)
	ev.AddField(namespacedKey, val)
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

// readResponses pulls from the response queue and spits them to STDOUT for
// debugging
func readResponses(responses chan libhoney.Response) {
	for r := range responses {
		var metadata string
		if r.Metadata != nil {
			metadata = fmt.Sprintf("%s", r.Metadata)
		}
		if r.StatusCode >= 200 && r.StatusCode < 300 {
			message := "Successfully sent event to Honeycomb"
			if metadata != "" {
				message += fmt.Sprintf(": %s", metadata)
			}
			fmt.Printf("%s\n", message)
		} else {
			fmt.Printf("Error sending event to Honeycomb! %s had code %d, err %v and response body %s \n",
				metadata, r.StatusCode, r.Err, r.Body)
		}
	}
}
