package beeline

import (
	"context"
	"fmt"
	"testing"

	"github.com/honeycombio/libhoney-go/transmission"

	libhoney "github.com/honeycombio/libhoney-go"
	"github.com/stretchr/testify/assert"
)

// TestNestedSpans tests that if you open and close several spans in the same
// function that fields added after the inner spans have closed are correctly
// added to the outer spans.  If you don't keep the context from sending the
// spans or somehow break re-inserting the parent span into the context after
// sending a child span, this test will fail.
func TestNestedSpans(t *testing.T) {
	mo := &transmission.MockSender{}
	client, err := libhoney.NewClient(libhoney.ClientConfig{
		APIKey:       "placeholder",
		Dataset:      "placeholder",
		APIHost:      "placeholder",
		Transmission: mo})
	assert.Equal(t, nil, err)
	Init(Config{Client: client})
	ctxroot, spanroot := StartSpan(context.Background(), "start")
	AddField(ctxroot, "start_col", 1)
	ctxmid, spanmid := StartSpan(ctxroot, "middle")
	AddField(ctxmid, "mid_col", 1)
	ctxleaf, spanleaf := StartSpan(ctxmid, "leaf")
	AddField(ctxleaf, "leaf_col", 1)
	spanleaf.Send()                       // sending leaf span
	AddField(ctxmid, "after_mid_col", 1)  // adding to middle span
	spanmid.Send()                        // sending middle span
	AddField(ctxroot, "end_start_col", 1) // adding to start span
	spanroot.Send()                       // sending start span

	events := mo.Events()
	assert.Equal(t, 3, len(events), "should have sent 3 events")
	var foundStart, foundMiddle bool
	for _, ev := range events {
		fields := ev.Data
		if fields["app.start_col"] == 1 {
			foundStart = true
			assert.Equal(t, fields["app.end_start_col"], 1, "ending start field should be in start span")
		}
		if fields["app.mid_col"] == 1 {
			foundMiddle = true
			assert.Equal(t, fields["app.after_mid_col"], 1, "after middle field should be in middle span")
		}
	}
	assert.True(t, foundStart, "didn't find the start span")
	assert.True(t, foundMiddle, "didn't find the middle span")
}

// TestBasicSpanAttributes verifies that creating and sending a span gives it
// all the basic required attributes: duration, trace, span, and parentIDs, and
// name.
func TestBasicSpanAttributes(t *testing.T) {
	mo := &transmission.MockSender{}
	client, err := libhoney.NewClient(libhoney.ClientConfig{
		APIKey:       "placeholder",
		Dataset:      "placeholder",
		APIHost:      "placeholder",
		Transmission: mo})
	assert.Equal(t, nil, err)
	Init(Config{Client: client})
	ctx, span := StartSpan(context.Background(), "start")
	AddField(ctx, "start_col", 1)
	ctxLeaf, spanLeaf := StartSpan(ctx, "leaf")
	AddField(ctxLeaf, "leaf_col", 1)
	spanLeaf.Send()
	span.Send()

	events := mo.Events()
	assert.Equal(t, 2, len(events), "should have sent 2 events")

	var foundRoot bool
	for _, ev := range events {
		fields := ev.Data
		name, ok := fields["name"]
		assert.True(t, ok, "failed to find name")
		_, ok = fields["duration_ms"]
		assert.True(t, ok, "failed to find duration_ms")
		_, ok = fields["trace.trace_id"]
		assert.True(t, ok, fmt.Sprintf("failed to find trace ID for span %s", name))
		_, ok = fields["trace.span_id"]
		assert.True(t, ok, fmt.Sprintf("failed to find span ID for span %s", name))

		spanType, ok := fields["meta.span_type"]
		if ok {
			spanTypeStr, ok := spanType.(string)
			assert.True(t, ok, "span field meta.span_type should be string")
			if spanTypeStr == "root" {
				foundRoot = true
			}
		} else {
			// non-root spans should have a parent ID
			_, ok = fields["trace.parent_id"]
			assert.True(t, ok, fmt.Sprintf("failed to find parent ID for span %s", name))
		}
		// root span will be missing parent ID
	}
	assert.True(t, foundRoot, "root span missing")
}
