package beeline

import (
	"context"
	"fmt"
	"os"

	internal "github.com/honeycombio/beeline-go/internal"
	"github.com/honeycombio/beeline-go/internal/sample"
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
	// SamplerHook is a function that will get run with the contents of each
	// event just before sending the event to Honeycomb. Register a function
	// with this config option to have manual control over sampling within the
	// beeline. Runs before the PresendHook.
	SamplerHook func(map[string]interface{}) (bool, int)
	// PresendHook is a function call that will get run with the contents of
	// each event just before sending them to Honeycomb. The function registered
	// here may mutate the map passed in to add, change, or drop fields from the
	// event before it gets sent to Honeycomb. Runs after the SamplerHook.
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
		internal.GlobalConfig.SamplerHook = config.SamplerHook
	} else {
		// configure and set a global sampler so sending traces can use it without
		// threading it through
		sampler, err := sample.NewDeterministicSampler(config.SampleRate)
		if err == nil {
			sample.GlobalSampler = sampler
		}
	}

	if config.PresendHook != nil {
		internal.GlobalConfig.PresendHook = config.PresendHook
	}
	return
}

// Flush sends any pending events to Honeycomb. This is optional; events will be
// flushed on a timer otherwise. It is useful to flush before AWS Lambda
// functions finish to ensure events get sent before AWS freezes the function.
// Flush implicitly ends all currently active spans.
func Flush(ctx context.Context) {
	trace := internal.GetTraceFromContext(ctx)
	trace.Send()
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
// middleware or wrapping a Handler, feel free to call AddFieldToSpan freely
// within your code. Pass it the context from the request (`r.Context()`) and
// the key and value you wish to add.This function is good for span-level data,
// eg timers or the arguments to a specific function call, etc. Fields added
// here are prefixed with `app.`
func AddFieldToSpan(ctx context.Context, key string, val interface{}) {
	namespacedKey := fmt.Sprintf("app.%s", key)
	span := internal.CurrentSpan(ctx)
	if span != nil {
		span.AddField(namespacedKey, val)
	}
}

// AddRollupFieldToSpan allows you to add a numeric field to the current span,
// and, when called on multiple spans within a trace, the sum of the field will
// be added to the root span. Use this when doing an action many times or on
// many spans and you want the sum of all those actions to be represented on the
// root span. Fields added here are prefixed with `app.` and the rolled up
// fields on the root span are prefixed with `totals.app.`
func AddRollupFieldToSpan(ctx context.Context, key string, val float64) {
	namespacedKey := fmt.Sprintf("app.%s", key)
	span := internal.CurrentSpan(ctx)
	if span != nil {
		span.AddRollupField(namespacedKey, val)
	}
}

// AddFieldToTrace adds the field to both the currently active span and all
// other spans involved in this trace that occur within this process.
// Additionally, these fields are packaged up and passed along to downstream
// processes if they are also using a beeline. This function is good for adding
// context that is better scoped to the request than this specific unit of work,
// eg user IDs, globally relevant feature flags, errors, etc. Fields added here
// are prefixed with `global.`
func AddFieldToTrace(ctx context.Context, key string, val interface{}) {
	namespacedKey := fmt.Sprintf("global.%s", key)
	trace := internal.GetTraceFromContext(ctx)
	if trace != nil {
		trace.AddField(namespacedKey, val)
	}
}

// HasTrace returns true if there is a trace in the current context
func HasTrace(ctx context.Context) bool {
	trace := internal.GetTraceFromContext(ctx)
	return trace != nil
}

// StartTraceWithIDs lets you start a trace with a specific set of IDs - it is
// used when you've received the IDs from another source (eg incoming HTTP
// headers). If you don't care what the IDs are, you may use either this or
// StartSpan to start a trace. You cannot change the trace IDs after a trace has
// begun. StartTraceWithIDs also starts a span (the root span) for this trace.
// You should call FinishSpan to close the span (and trace) started by this
// function.
func StartTraceWithIDs(ctx context.Context, traceID, parentID, name string) context.Context {
	ctx, _ = internal.StartTraceWithIDs(ctx, traceID, parentID, name)
	return ctx
}

// StartSpan lets you start a new span as a child of an already instrumented
// handler. If there isn't an existing wrapped handler in the context when this
// is called, it will start a new trace. Spans automatically get a `duration_ms`
// field when they are ended; you should not explicitly set the duration unless
// you want to override it. The name argument will be the primary way the span
// is identified in the trace view within Honeycomb. You must use the returned
// context to ensure attributes are added to the correct span.
func StartSpan(ctx context.Context, name string) context.Context {
	ctx, _ = internal.StartSpan(ctx, name)
	return ctx
}

// TODO do we need to provide some way to extract the trace / span from this
// context and shove it in to another? I suspect strange things would happen to
// any downstream services that want to use the context for timeouts or
// cancellation if the request finished (which triggers the original context
// getting cancelled, which is most likely inappropriate for an async span).

// StartAsyncSpan is different from StartSpan in that when finishing a trace, it
// does not get automatically sent. When finishing an async span it gets sent
// immediately.
func StartAsyncSpan(ctx context.Context, name string) context.Context {
	ctx, _ = internal.StartAsyncSpan(ctx, name)
	return ctx
}

// FinishSpan finishes the currently active span. It should only be called for
// spans created with StartTraceWithIDs, StartSpan, or StartAsyncSpan. Use the
// context returned by finish span in order to ensure operations on the
// "current" span continue to work correctly. The only time you should ignore
// the context returned by this function is when using it as a defer to finish a
// span also started in this scope; in that case when the function finishes the
// previously scoped context will still have the right span marked as current.
func FinishSpan(ctx context.Context) context.Context {
	return internal.FinishSpan(ctx)
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
