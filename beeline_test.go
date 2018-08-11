package beeline

import (
	"context"
	"testing"

	libhoney "github.com/honeycombio/libhoney-go"
	"github.com/stretchr/testify/assert"
)

// TestNestedSpans tests that if you open and close several spans in the same
// function that fields added after the inner spans have closed are correctly
// added to the outer spans.  If you don't keep the context from finishing the
// spans or somehow break re-inserting the parent span into the context after
// finishing a child span, this test will fail.
func TestNestedSpans(t *testing.T) {
	mo := &libhoney.MockOutput{}
	libhoney.Init(
		libhoney.Config{
			APIHost:  "placeholder",
			WriteKey: "placeholder",
			Dataset:  "placeholder",
			Output:   mo,
		},
	)
	ctx := StartSpan(context.Background(), "start")
	AddField(ctx, "start_col", 1)
	ctx = StartSpan(ctx, "middle")
	AddField(ctx, "mid_col", 1)
	ctx = StartSpan(ctx, "leaf")
	AddField(ctx, "leaf_col", 1)
	ctx = FinishSpan(ctx)             // finishing leaf span
	AddField(ctx, "after_mid_col", 1) // adding to middle span
	ctx = FinishSpan(ctx)             // finishing middle span
	AddField(ctx, "end_start_col", 1) // adding to start span
	ctx = FinishSpan(ctx)             // finishing start span
	Flush(ctx)

	events := mo.Events()
	assert.Equal(t, 3, len(events), "should have sent 3 events")
	var foundStart, foundMiddle bool
	for _, ev := range events {
		fields := ev.Fields()
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
