package beeline

import (
	"context"
	"fmt"
	"os"

	"github.com/honeycombio/beeline-go/sample"
	"github.com/honeycombio/beeline-go/trace"
	libhoney "github.com/honeycombio/libhoney-go"
)

const (
	defaultWriteKey   = "apikey-placeholder"
	defaultDataset    = "beeline-go"
	defaultSampleRate = 1
)

// Config is the place where you configure your Honeycomb write key and dataset
// name. WriteKey is the only required field in order to actually send events to
// Honeycomb.
type Config struct {
	// Writekey is your Honeycomb authentication token, available from
	// https://ui.honeycomb.io/account. default: apikey-placeholder
	WriteKey string
	// Dataset is the name of the Honeycomb dataset to which events will be
	// sent. default: beeline-go
	Dataset string
	// Service Name identifies your application. While optional, setting this
	// field is extremely valuable when you instrument multiple services. If set
	// it will be added to all events as `service_name`
	ServiceName string
	// SamplRate is a positive integer indicating the rate at which to sample
	// events. Default sampling is at the trace level - entire traces will be
	// kept or dropped. default: 1 (meaning no sampling)
	SampleRate uint
	// SamplerHook is a function that will get run with the contents of each
	// event just before sending the event to Honeycomb. Register a function
	// with this config option to have manual control over sampling within the
	// beeline. The function should return true if the event should be kept and
	// false if it should be dropped.  If it should be kept, the returned
	// integer is the sample rate that has been applied. The SamplerHook
	// overrides the default sampler. Runs before the PresendHook.
	SamplerHook func(map[string]interface{}) (bool, int)
	// PresendHook is a function call that will get run with the contents of
	// each event just before sending them to Honeycomb. The function registered
	// here may mutate the map passed in to add, change, or drop fields from the
	// event before it gets sent to Honeycomb. Does not get invoked if the event
	// is going to be dropped because of sampling. Runs after the SamplerHook.
	PresendHook func(map[string]interface{})
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
		WriteKey: config.WriteKey,
		Dataset:  config.Dataset,
		Output:   output,
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

	// Use the sampler hook if it's defined, otherwise a deterministic sampler
	if config.SamplerHook != nil {
		trace.GlobalConfig.SamplerHook = config.SamplerHook
	} else {
		// configure and set a global sampler so sending traces can use it
		// without threading it through
		sampler, err := sample.NewDeterministicSampler(config.SampleRate)
		if err == nil {
			sample.GlobalSampler = sampler
		}
	}

	if config.PresendHook != nil {
		trace.GlobalConfig.PresendHook = config.PresendHook
	}
	return
}

// Flush sends any pending events to Honeycomb. This is optional; events will be
// flushed on a timer otherwise. It is useful to flush before AWS Lambda
// functions finish to ensure events get sent before AWS freezes the function.
// Flush implicitly ends all currently active spans.
func Flush(ctx context.Context) {
	tr := trace.GetTraceFromContext(ctx)
	if tr != nil {
		tr.Send()
	}
	libhoney.Flush()
}

// Close shuts down the beeline. Closing does not send any pending traces but
// does flush any pending libhoney events and blocks until they have been sent.
// It is optional to close the beeline, and prohibited to try and send an event
// after the beeline has been closed.
func Close() {
	libhoney.Close()
}

// AddField allows you to add a single field to an event anywhere downstream of
// an instrumented request. After adding the appropriate middleware or wrapping
// a Handler, feel free to call AddField freely within your code. Pass it the
// context from the request (`r.Context()`) and the key and value you wish to
// add.This function is good for span-level data, eg timers or the arguments to
// a specific function call, etc. Fields added here are prefixed with `app.`
func AddField(ctx context.Context, key string, val interface{}) {
	span := trace.GetSpanFromContext(ctx)
	if span != nil {
		if val != nil {
			namespacedKey := fmt.Sprintf("app.%s", key)
			if valErr, ok := val.(error); ok {
				// treat errors specially because it's a pain to have to
				// remember to stringify them
				span.AddField(namespacedKey, valErr.Error())
			} else {
				span.AddField(namespacedKey, val)
			}
		}
	}
}

// AddFieldToTrace adds the field to both the currently active span and all
// other spans involved in this trace that occur within this process.
// Additionally, these fields are packaged up and passed along to downstream
// processes if they are also using a beeline. This function is good for adding
// context that is better scoped to the request than this specific unit of work,
// eg user IDs, globally relevant feature flags, errors, etc. Fields added here
// are prefixed with `app.`
func AddFieldToTrace(ctx context.Context, key string, val interface{}) {
	namespacedKey := fmt.Sprintf("app.%s", key)
	tr := trace.GetTraceFromContext(ctx)
	if tr != nil {
		tr.AddField(namespacedKey, val)
	}
}

// StartSpan lets you start a new span as a child of an already instrumented
// handler. If there isn't an existing wrapped handler in the context when this
// is called, it will start a new trace. Spans automatically get a `duration_ms`
// field when they are ended; you should not explicitly set the duration. The
// name argument will be the primary way the span is identified in the trace
// view within Honeycomb. You get back a fresh context with the new span in it
// as well as the actual span that was just created. You should call
// `span.Send()` when the span should be sent (often in a defer immediately
// after creation). You should pass the returned context downstream.
func StartSpan(ctx context.Context, name string) (context.Context, *trace.Span) {
	span := trace.GetSpanFromContext(ctx)
	var newSpan *trace.Span
	if span != nil {
		ctx, newSpan = span.CreateChild(ctx)
	} else {
		// there is no trace active; we should make one, but use the root span
		// as the "new" span instead of creating a child of this mostly empty
		// span
		ctx, _ = trace.NewTrace(ctx, "")
		newSpan = trace.GetSpanFromContext(ctx)
	}
	newSpan.AddField("name", name)
	return ctx, newSpan
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
