package beeline

import (
	"context"
	"fmt"
	"os"

	internal "github.com/honeycombio/beeline-go/internal"
	libhoney "github.com/honeycombio/libhoney-go"
)

const (
	defaultWriteKey   = "writekey-placeholder"
	defaultDataset    = "beeline-go"
	defaultSampleRate = 1
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
// Flush implicitly ends all currently active spans.
func Flush() {
	libhoney.Flush()
}

// Close shuts down the beeline. Closing flushes any pending events and blocks
// until they have been sent. It is optional to close the beeline, and
// prohibited to try and send an event after the beeline has been closed.
// Close implicitly ends all currently active spans.
func Close() {
	libhoney.Close()
}

// AddField (Deprecated as of 0.2.0) is synonymous with AddFieldToSpan. It adds
// the current field to the currently active span.
func AddField(ctx context.Context, key string, val interface{}) {
	AddFieldToSpan(ctx, key, val)
}

// AddFieldToSpan allows you to add a single field to an event anywhere
// downstream of an instrumented request. After adding the appropriate
// middleware or wrapping a Handler, feel free to call AddField freely within
// your code. Pass it the context from the request (`r.Context()`) and the key
// and value you wish to add.This function is good for span-level data, eg
// timers or the arguments to a specific function call, etc..
func AddFieldToSpan(ctx context.Context, key string, val interface{}) {
	namespacedKey := fmt.Sprintf("app.%s", key)
	internal.CurrentSpan(ctx).AddField(namespacedKey, val)
}

// AddRollupFieldToSpan allows you to add a numeric field to the current span,
// and, when called on multiple spans within a trace, the sum of the field will
// be added to the root span
func AddRollupFieldToSpan(ctx context.Context, key string, val float64) {
	namespacedKey := fmt.Sprintf("app.%s", key)
	internal.GetTraceFromContext(ctx).AddRollupField(namespacedKey, val)
}

// AddFieldToTrace adds the field to both the currently active span and any
// other spans involved in this trace that occur within this process.  This
// function is good for adding context that is better scoped to the request than
// this specific unit of work, eg user IDs, globally relevant feature flags,
// errors, etc. Note that these values do not currently traverse process
// boundaries.
func AddFieldToTrace(ctx context.Context, key string, val interface{}) {
	namespacedKey := fmt.Sprintf("global.%s", key)
	internal.GetTraceFromContext(ctx).AddField(namespacedKey, val)
}

// HasTrace returns true if there is a trace in the current context
func HasTrace(ctx context.Context) bool {
	trace := internal.GetTraceFromContext(ctx)
	return trace != nil
}

// StartSpan lets you start a new span as a child of an already instrumented
// handler. Use the returned contexts for all future calls to AddField to ensure
// they're added to the right span. If there isn't an existing wrapped handler
// in the context when this is called, it will start a new trace. Spans
// automatically get a `duration_ms` field when they are ended; you should not
// explicitly set the duration unless you want to override it.
func StartSpan(ctx context.Context) context.Context {
	return internal.PushSpanOnStack(ctx)
}

// SetTraceIDs lets you override the generated trace ID with IDs you've received
// from another source (eg incoming HTTP headers)
func SetTraceIDs(ctx context.Context, traceID, parentID string) {
	t := internal.GetTraceFromContext(ctx)
	t.SetTraceIDs(traceID, parentID)
}

// EndSpan finishes the currently active span. It should only be called for
// spans created with StartSpan
func EndSpan(ctx context.Context) {
	internal.EndSpan(ctx)
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
