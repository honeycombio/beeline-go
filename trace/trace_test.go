package trace

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/honeycombio/beeline-go/client"
	"github.com/honeycombio/beeline-go/propagation"
	libhoney "github.com/honeycombio/libhoney-go"
	"github.com/honeycombio/libhoney-go/transmission"
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

	t.Run("Serializing headers does not race with adding trace level fields", func(t *testing.T) {
		wg := &sync.WaitGroup{}
		wg.Add(2)
		go func() {
			spFromContext.AddTraceField("race", "hope not")
			wg.Done()
		}()
		go func() {
			spFromContext.SerializeHeaders()
			wg.Done()
		}()
		wg.Wait()
	})
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
		evName := ev.Data["name"].(string)

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

	_, childSpan := span.CreateChild(ctx)
	assert.Equal(t, span.spanID, childSpan.parentID, "child span's parent ID should be parent's span ID")
	assert.Equal(t, span, childSpan.parent, "child span should have a pointer to parent")
	assert.Len(t, span.children, 1, "parent span should have a child")
	assert.Equal(t, childSpan, span.children[0], "parent span's child should be child span")

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

	// add some rollup fields
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
	expectedHeader := fmt.Sprintf("1;trace_id=%s,parent_id=%s,dataset=placeholder,context=eyJ0cjEiOiJ2cjEifQ==", tr.traceID, span.spanID)
	assert.Equal(t, expectedHeader, headers, "serialized span should match expectations")

	childSpan.Send()

	assert.True(t, childSpan.isSent, "child span should now be sent")
	assert.Len(t, span.children, 0, "child span should now be removed from the parent span's children since it's been sent")

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
	assert.Equal(t, 4, len(events), "should have sent 4 events, rs, c1, and c2")
	var foundRoot, foundSpan, foundAsync int
	for _, ev := range events {
		// some things should be true for all spans
		assert.IsType(t, float64(0), ev.Data["duration_ms"], "span should have a numeric duration")
		assert.Equal(t, "vr1", ev.Data["tr1"], "span should have trace level field")

		// a few things are different on each of the three span types
		switch ev.Data["meta.span_type"].(string) {
		case "root":
			foundRoot++
			assert.Nil(t, ev.Data["trace.parent_id"], "root span should have no parent ID")
			assert.Equal(t, float64(12), ev.Data["rollup.r1"], "root span should have rolled up fields")
		case "async":
			foundAsync++
		case "leaf":
			foundSpan++
		default:
			t.Error("unexpected event found")
		}
	}
	assert.Equal(t,
		[]int{1, 1, 2},
		[]int{foundRoot, foundAsync, foundSpan},
		"all four spans should be sent")

}

func TestCreateAsyncSpanDoesNotCauseRaceInSend(t *testing.T) {
	setupLibhoney()
	ctx, tr := NewTrace(context.Background(), t.Name())
	rs := tr.GetRootSpan()

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		rs.Send()
		wg.Done()
	}()
	go func() {
		rs.CreateAsyncChild(ctx)
		wg.Done()
	}()
	wg.Wait()
}

// TestCreateSubSpanDoesNotCauseRaceInSend exists because when sending a root
// span, you don't consider the length of the children to determine if the span
// is a leaf. By doing the same test as above on a subspan, we test for a race
// on the number of chilrden in a span.
func TestCreateSubSpanDoesNotCauseRaceInSend(t *testing.T) {
	setupLibhoney()
	ctx, tr := NewTrace(context.Background(), t.Name())
	rs := tr.GetRootSpan()
	ctx, subsp := rs.CreateChild(ctx)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		subsp.Send()
		wg.Done()
	}()
	go func() {
		subsp.CreateChild(ctx)
		wg.Done()
	}()
	wg.Wait()
}

func TestChildAndParentSendsDoNotRace(t *testing.T) {
	setupLibhoney()

	wg := &sync.WaitGroup{}
	wg.Add(10)
	for i := 0; i < 5; i++ {
		ctx, tr := NewTrace(context.Background(), t.Name())
		rs := tr.GetRootSpan()

		go func() {
			rs.Send()
			wg.Done()
		}()
		go func() {
			ct, sp1 := rs.CreateChild(ctx)
			ct, sp2 := sp1.CreateChild(ct)
			ct, sp3 := sp2.CreateChild(ct)
			sp1.Send()
			sp2.Send()
			sp3.Send()

			wg.Done()
		}()
	}

	wg.Wait()
}

func TestAddFieldDoesNotCauseRaceInSendHooks(t *testing.T) {
	samplerHook := func(fields map[string]interface{}) (bool, int) {
		for range fields {
			// do nothing, we just want to iterate
		}

		return true, 1
	}

	setupLibhoney()
	GlobalConfig.SamplerHook = samplerHook
	defer func() {
		GlobalConfig.SamplerHook = nil
	}()

	run := make(chan *Span)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		for s := range run {
			time.Sleep(time.Microsecond)
			s.Send()
		}
		wg.Done()
	}()

	ctx, tr := NewTrace(context.Background(), "")
	rs := tr.GetRootSpan()

	for i := 0; i < 100; i++ {
		_, s := rs.CreateChild(ctx)
		s.AddField("a", i)
		run <- s
		time.Sleep(time.Microsecond)
		for j := 0; j < 100; j++ {
			s.AddField("b", j)
		}
	}
	close(run)
	// need to wait here to avoid a race on resetting the SamplerHook
	wg.Wait()
}

func TestPropagatedFields(t *testing.T) {
	prop := &propagation.Propagation{
		TraceID:  "abcdef123456",
		ParentID: "0102030405",
		Dataset:  "imadataset",
		TraceContext: map[string]interface{}{
			"userID":   float64(1),
			"errorMsg": "failed to sign on",
			"toRetry":  true,
		},
	}
	serial := propagation.MarshalTraceContext(prop)
	ctx, tr := NewTrace(context.Background(), serial)

	assert.NotNil(t, tr.builder, "traces should have a builder")
	assert.Equal(t, prop.TraceID, tr.traceID, "trace id should have propagated")
	assert.Equal(t, prop.ParentID, tr.parentID, "parent id should have propagated")
	assert.Equal(t, prop.Dataset, tr.builder.Dataset, "dataset should have propagated")
	assert.Equal(t, prop.TraceContext, tr.traceLevelFields, "trace fields should have propagated")

	trFromContext := GetTraceFromContext(ctx)
	assert.Equal(t, tr, trFromContext, "new trace should put the trace in the context")

	_, tr2 := NewTrace(context.Background(), tr.GetRootSpan().SerializeHeaders())
	assert.Equal(t, tr.traceID, tr2.traceID, "trace ID should shave propagated")
	assert.NotEqual(t, tr.parentID, tr2.parentID, "parent ID should have changed")
	assert.Equal(t, tr.builder.Dataset, tr2.builder.Dataset, "dataset should have propagated")
	assert.Equal(t, tr.traceLevelFields, tr2.traceLevelFields, "trace fields should have propagated")

	prop = &propagation.Propagation{
		Dataset: "imadataset",
		TraceContext: map[string]interface{}{
			"userID": float64(1),
		},
	}
	serial = propagation.MarshalTraceContext(prop)
	ctx, tr = NewTrace(context.Background(), serial)
	assert.NotNil(t, tr.builder, "traces should have a builder")
	assert.NotEqual(t, "", tr.traceID, "trace id should have propagated")
	assert.Equal(t, "", tr.parentID, "parent id should have propagated")
	assert.Equal(t, prop.Dataset, tr.builder.Dataset, "dataset should have propagated")
	assert.Equal(t, prop.TraceContext, tr.traceLevelFields, "trace fields should have propagated")

	ctx, tr = NewTrace(context.Background(), "garbage")
	assert.NotNil(t, tr.builder, "traces should have a builder")
	assert.NotEqual(t, "", tr.traceID, "trace id should have propagated")
	assert.Equal(t, "", tr.parentID, "parent id should have propagated")
	assert.Equal(t, "placeholder", tr.builder.Dataset, "dataset should have propagated")
	assert.Equal(t, map[string]interface{}{}, tr.traceLevelFields, "trace fields should have propagated")

}

// BenchmarkSendChildSpans benchmarks creating and sending child spans in
// parallel. We do a good bit of locking when spans are sent and we want to
// check to ensure that we don't regress the performance of sending spans.
func BenchmarkSendChildSpans(b *testing.B) {
	setupLibhoney()
	ctx, tr := NewTrace(context.Background(), b.Name())
	rs := tr.GetRootSpan()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, s := rs.CreateChild(ctx)
			s.Send()
		}
	})
}

func setupLibhoney() *transmission.MockSender {
	mo := &transmission.MockSender{}
	c, _ := libhoney.NewClient(
		libhoney.ClientConfig{
			APIKey:       "placeholder",
			Dataset:      "placeholder",
			APIHost:      "placeholder",
			Transmission: mo,
		})
	client.Set(c)

	return mo
}
