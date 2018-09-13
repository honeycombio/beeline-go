package trace

import (
	"context"
	"fmt"
	"testing"

	libhoney "github.com/honeycombio/libhoney-go"
	"github.com/stretchr/testify/assert"
)

// TestNewTrace create traces and make sure they're populated with all the
// expected things
func TestNewTrace(t *testing.T) {
	// test basic new trace
	ctx, tr := NewTrace(context.Background(), "")
	assert.NotNil(t, tr.builder, "traces should have a builder")
	assert.NotEmpty(t, tr.traceID, "trace should have a trace ID")
	assert.Empty(t, tr.parentID, "trace created with no headers should have an empty parent ID")
	assert.NotNil(t, tr.rollupFields, "trace should initialize rollup fields map")
	assert.NotNil(t, tr.rootSpan, "trace should have a root span")
	assert.NotNil(t, tr.traceLevelFields, "trace should initialize trace level fields map")
	trFromContext := GetTraceFromContext(ctx)
	assert.Equal(t, tr, trFromContext, "new trace should put the trace in the context")
	spFromContext := GetSpanFromContext(ctx)
	assert.Equal(t, tr.rootSpan, spFromContext, "new trace should put the root span in the context")

	// trace created with headers should take the trace and parent IDs and context
	// serialized header with IDs and three fields in the context:
	// 1;trace_id=abcdef123456,parent_id=0102030405,context=eyJlcnJvck1zZyI6ImZhaWxlZCB0byBzaWduIG9uIiwidG9SZXRyeSI6dHJ1ZSwidXNlcklEIjoxfQ==
	// three fields are {"userID":1,"errorMsg":"failed to sign on","toRetry":true}
	// string taken from propagation_test.go
	serializedHeaders := "1;trace_id=abcdef123456,parent_id=0102030405,context=eyJlcnJvck1zZyI6ImZhaWxlZCB0byBzaWduIG9uIiwidG9SZXRyeSI6dHJ1ZSwidXNlcklEIjoxfQ=="
	_, tr = NewTrace(context.Background(), serializedHeaders)
	assert.Equal(t, "abcdef123456", tr.traceID, "trace with headers should take trace ID")
	assert.Equal(t, "0102030405", tr.parentID, "trace with headers should take parent ID")
	assert.Equal(t, float64(1), tr.traceLevelFields["userID"], "trace with headers should populate trace level fields")
	assert.Equal(t, "failed to sign on", tr.traceLevelFields["errorMsg"], "trace with headers should populate trace level fields")
	assert.Equal(t, true, tr.traceLevelFields["toRetry"], "trace with headers should populate trace level fields")
}

// TestAddField tests adding a field to a trace
func TestAddField(t *testing.T) {
	_, tr := NewTrace(context.Background(), "")
	tr.AddField("wander", "lust")
	assert.Equal(t, "lust", tr.traceLevelFields["wander"], "AddField on a trace should add the field to the trace level fields map")
}

// TestRollupField tests adding a field to a trace
func TestRollupField(t *testing.T) {
	_, tr := NewTrace(context.Background(), "")
	tr.addRollupField("bignum", 5)
	tr.addRollupField("bignum", 5)
	tr.addRollupField("smallnum", 0.1)
	assert.Equal(t, float64(10), tr.rollupFields["bignum"], "addRollupField on a trace should sum the fields added")
	assert.Equal(t, 0.1, tr.rollupFields["smallnum"], "addRollupField on a trace should sum the fields added")
}

// TestGetRootSpan verifies the real root span is returned
func TestGetRootSpan(t *testing.T) {
	_, tr := NewTrace(context.Background(), "")
	sp := tr.GetRootSpan()
	assert.Equal(t, tr.rootSpan, sp, "get root span should return the trace's root span")
}

// TestSendTrace should verify that sending a trace calls send on all
// synchronous children
func TestSendTrace(t *testing.T) {
	mo := setupLibhoney()
	ctx, tr := NewTrace(context.Background(), "")
	rs := tr.GetRootSpan()
	rs.AddField("name", "rs")
	ctx, c1 := rs.CreateChild(ctx)
	c1.AddField("name", "c1")
	ctx, c2 := c1.CreateChild(ctx)
	c2.AddField("name", "c2")
	ctx, ac1 := c1.CreateAsyncChild(ctx)
	ac1.AddField("name", "ac1")
	// synchronous children of asynchronous spans get sent by themselves or the
	// async parent but *not* by the async's parent
	ctx, notSentChild := ac1.CreateChild(ctx)
	notSentChild.AddField("name", "notSentChild")

	// send the trace. expect rs, c1, and c2 to get sent. expect ac1 and
	// notSentChild to not get sent
	tr.Send()

	// expected maps name to whether it got sent (true - got sent, false, did not)
	expected := map[string]bool{
		"rs":           true,
		"c1":           true,
		"c2":           true,
		"ac1":          false,
		"notSentChild": false,
	}
	// initialize actual to all false aka "didn't find any"
	actual := map[string]bool{
		"rs":           false,
		"c1":           false,
		"c2":           false,
		"ac1":          false,
		"notSentChild": false,
	}
	// look through the fields in the mock libhoney and verify we got each
	events := mo.Events()
	assert.Equal(t, 3, len(events), "should have sent 3 events, rs, c1, and c2")
	for _, ev := range events {
		evName := ev.Fields()["name"].(string)

		actual[evName] = true
	}
	assert.Equal(t, expected, actual, "actually sent events doesn't match expectations")
}

// TestCreateSpan verifies spans created have the expected basic contents
func TestSpan(t *testing.T) {
	mo := setupLibhoney()

	ctx, tr := NewTrace(context.Background(), "")
	rs := tr.GetRootSpan()

	ctx, span := rs.CreateChild(ctx)
	assert.Equal(t, false, span.isAsync, "regular span should not be async")
	assert.Equal(t, false, span.IsAsync(), "regular span should not be async")
	assert.Equal(t, false, span.isSent, "regular span should not yet be sent")
	assert.Equal(t, false, span.isRoot, "regular span should not be root")
	assert.Equal(t, true, rs.isRoot, "root span should be root")
	assert.Equal(t, span, rs.children[0], "root span's first child should be span")
	assert.NotNil(t, span.ev, "span should have an embedded event")
	assert.Equal(t, rs.spanID, span.parentID, "span's parent ID should be parent's span ID")
	assert.Equal(t, rs, span.parent, "span should have a pointer to parent")
	assert.NotNil(t, span.rollupFields, "span should have an initialized rollupFields map")
	assert.NotNil(t, span.timer, "span should have an initialized timer")
	assert.Equal(t, tr, span.trace, "span should have a pointer to trace")

	ctx, asyncSpan := rs.CreateAsyncChild(ctx)
	assert.Equal(t, true, asyncSpan.isAsync, "async span should not be async")
	assert.Equal(t, true, asyncSpan.IsAsync(), "async span should not be async")
	assert.Equal(t, false, asyncSpan.isSent, "async span should not yet be sent")
	assert.Equal(t, false, asyncSpan.isRoot, "async span should not be root")
	assert.Equal(t, true, rs.isRoot, "root span should be root")
	assert.Equal(t, asyncSpan, rs.children[1], "root span's second child should be asyncSpan")
	assert.NotNil(t, asyncSpan.ev, "span should have an embedded event")
	assert.Equal(t, rs.spanID, asyncSpan.parentID, "span's parent ID should be parent's span ID")
	assert.Equal(t, rs, asyncSpan.parent, "span should have a pointer to parent")
	assert.NotNil(t, asyncSpan.rollupFields, "span should have an initialized rollupFields map")
	assert.NotNil(t, asyncSpan.timer, "span should have an initialized timer")
	assert.Equal(t, tr, asyncSpan.trace, "span should have a pointer to trace")

	span.AddField("f1", "v1")
	assert.Equal(t, "v1", span.ev.Fields()["f1"].(string), "after adding a field, field should exist on the span")

	span.AddRollupField("r1", 2)
	span.AddRollupField("r1", 3)
	asyncSpan.AddRollupField("r1", 7)
	assert.Equal(t, float64(5), span.rollupFields["r1"], "repeated rollup fields should be summed on the span")
	assert.Equal(t, float64(7), asyncSpan.rollupFields["r1"], "rollup fields should remain separate on separate spans")
	assert.Equal(t, float64(12), tr.rollupFields["r1"], "rollup fields should have the grand total in the trace")

	chillins := rs.GetChildren()
	assert.Equal(t, rs.children, chillins, "get children should return the actual slice of children")
	spanParent := span.GetParent()
	asyncParent := asyncSpan.GetParent()
	assert.Equal(t, spanParent, asyncParent, "span and asyncSpan should have the same parent")
	assert.Equal(t, rs, asyncParent, "span and asyncSpan's parent should be the root span")

	span.AddTraceField("tr1", "vr1")
	assert.Equal(t, "vr1", tr.traceLevelFields["tr1"], "span's trace fields should be added to the trace")
	assert.Nil(t, span.ev.Fields()["tr1"], "span should not have trace fields present")

	headers := span.SerializeHeaders()
	// magical string here is base64 encoded "tr1" field for the trace propagation
	expectedHeader := fmt.Sprintf("1;trace_id=%s,parent_id=%s,context=eyJ0cjEiOiJ2cjEifQ==", tr.traceID, span.spanID)
	assert.Equal(t, expectedHeader, headers, "serialized span should match expectations")

	// sending the root span should send span too
	rs.Send()

	assert.Equal(t, true, rs.isSent, "root span should now be sent")
	assert.Equal(t, true, span.isSent, "regular span should now be sent")
	assert.Equal(t, false, asyncSpan.isSent, "async span should not yet be sent")

	asyncSpan.Send()

	assert.Equal(t, true, span.isSent, "regular span should now be sent")
	assert.Equal(t, true, asyncSpan.isSent, "async span should not yet be sent")

	// ok go through the actually sent events and check some stuff
	events := mo.Events()
	assert.Equal(t, 3, len(events), "should have sent 3 events, rs, c1, and c2")
	var foundRoot, foundSpan, foundAsync bool
	for _, ev := range events {
		// some things should be true for all spans
		assert.IsType(t, float64(0), ev.Fields()["duration_ms"], "span should have a numeric duration")
		assert.Equal(t, "vr1", ev.Fields()["tr1"], "span should have trace level field")

		// a few things are different on each of the three span types
		switch ev.Fields()["meta.span_type"].(string) {
		case "root":
			foundRoot = true
			assert.Nil(t, ev.Fields()["trace.parent_id"], "root span should have no parent ID")
		case "async":
			foundAsync = true
		case "leaf":
			foundSpan = true
		default:
			t.Error("unexpected event found")
		}
	}
	assert.Equal(t,
		[]bool{true, true, true},
		[]bool{foundRoot, foundAsync, foundSpan},
		"all three spans should be sent")

}

func setupLibhoney() *libhoney.MockOutput {
	mo := &libhoney.MockOutput{}
	libhoney.Init(
		libhoney.Config{
			APIHost:  "placeholder",
			WriteKey: "placeholder",
			Dataset:  "placeholder",
			Output:   mo,
		},
	)
	return mo
}
