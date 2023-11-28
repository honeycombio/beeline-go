package beeline

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/honeycombio/libhoney-go/transmission"

	cmap "github.com/orcaman/concurrent-map/v2"

	libhoney "github.com/honeycombio/libhoney-go"
	"github.com/stretchr/testify/assert"
)

// TestNestedSpans tests that if you open and close several spans in the same
// function that fields added after the inner spans have closed are correctly
// added to the outer spans.  If you don't keep the context from sending the
// spans or somehow break re-inserting the parent span into the context after
// sending a child span, this test will fail.
func TestNestedSpans(t *testing.T) {
	mo := setupLibhoney(t)
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
	mo := setupLibhoney(t)
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

func BenchmarkCreateSpan(b *testing.B) {
	setupLibhoney(b)

	ctx, _ := StartSpan(context.Background(), "parent")
	for n := 0; n < b.N; n++ {
		StartSpan(ctx, "child")
	}
}

func BenchmarkBeelineAddField_PrefixedKey(b *testing.B) {
	setupLibhoney(b)

	ctx, _ := StartSpan(context.Background(), "parent")
	for n := 0; n < b.N; n++ {
		AddField(ctx, "app.foo", 1)
	}
}

func BenchmarkBeelineAddField_ConsistentKey(b *testing.B) {
	setupLibhoney(b)

	ctx, _ := StartSpan(context.Background(), "parent")
	for n := 0; n < b.N; n++ {
		AddField(ctx, "foo", 1)
	}
}

func BenchmarkBeelineAddField_InconsistentKey(b *testing.B) {
	setupLibhoney(b)

	ctx, _ := StartSpan(context.Background(), "parent")
	for n := 0; n < b.N; n++ {
		AddField(ctx, strconv.Itoa(n), 1)
	}
}

func setupLibhoney(t testing.TB) *transmission.MockSender {
	mo := &transmission.MockSender{}
	client, err := libhoney.NewClient(
		libhoney.ClientConfig{
			APIKey:       "placeholder",
			Dataset:      "placeholder",
			APIHost:      "placeholder",
			Transmission: mo,
		},
	)
	assert.Equal(t, nil, err)

	Init(Config{Client: client})

	return mo
}

func getRandomString() string {
	chars := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	length := rand.Intn(20) + 5
	result := make([]rune, length)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

func getPrefixedFieldNameOrig(name string) string {
	return "app." + name
}

var syncCache sync.Map

// sync.Map avoids a lot of locking overhead but has no size limit
func getPrefixedFieldNameSync(key string) string {
	const prefix = "app."

	// return if the key already has the prefix
	if strings.HasPrefix(key, prefix) {
		return key
	}

	// check the cache first
	val, ok := syncCache.Load(key)
	if ok {
		return val.(string)
	}

	// not in the cache, so add it
	prefixedKey := prefix + key
	syncCache.Store(key, prefixedKey)
	return prefixedKey
}

var concurrentMap cmap.ConcurrentMap[string, string]

func getPrefixedFieldNameConcurrent(key string) string {
	const prefix = "app."

	// return if the key already has the prefix
	if strings.HasPrefix(key, prefix) {
		return key
	}

	// check the cache first
	val, ok := concurrentMap.Get(key)
	if ok {
		return val
	}

	val = prefix + key
	concurrentMap.Set(key, val)

	return val
}

func getPrefixedFieldNameEjectRandom(key string) string {
	const prefix = "app."

	// return if the key already has the prefix
	if strings.HasPrefix(key, prefix) {
		return key
	}

	// check the cache using a read lock first
	cachedFieldNamesLock.RLock()
	val, ok := cachedFieldNames[key]
	cachedFieldNamesLock.RUnlock()
	if ok {
		return val
	}

	// not in the cache, so get a write lock
	cachedFieldNamesLock.Lock()
	defer cachedFieldNamesLock.Unlock()

	// check again in case it was added while we were waiting for the lock
	val, ok = cachedFieldNames[key]
	if ok {
		return val
	}

	// before we add the key to the cache, reset the cache if it's getting too big.
	// this can happen if lots of unique keys are being used and we don't want to
	// grow the cache indefinitely
	if len(cachedFieldNames) > 1000 {
		for k := range cachedFieldNames {
			delete(cachedFieldNames, k)
			break
		}
	}

	// add the prefixed key to the cache and return it
	prefixedKey := prefix + key
	cachedFieldNames[key] = prefixedKey
	return prefixedKey
}

func BenchmarkGetPrefixedFieldNameBasic(b *testing.B) {
	funcs := map[string]func(string) string{
		"orig":  getPrefixedFieldNameOrig,
		"new":   getPrefixedFieldName,
		"sync":  getPrefixedFieldNameSync,
		"conc":  getPrefixedFieldNameConcurrent,
		"eject": getPrefixedFieldNameEjectRandom,
	}
	for _, numFields := range []int{10, 100, 1000, 3000} {
		for name, f := range funcs {
			b.Run(fmt.Sprintf("%s-%d", name, numFields), func(b *testing.B) {
				names := make([]string, numFields)
				for i := 0; i < numFields; i++ {
					names[i] = getRandomString()
				}
				concurrentMap = cmap.New[string]()
				b.ResetTimer()
				for n := 0; n < b.N; n++ {
					f(names[rand.Intn(numFields)])
				}
			})
		}
	}
}

func BenchmarkGetPrefixedFieldNameParallel(b *testing.B) {
	funcs := map[string]func(string) string{
		"orig": getPrefixedFieldNameOrig,
		"new":  getPrefixedFieldName,
		"sync": getPrefixedFieldNameSync,
		"conc": getPrefixedFieldNameConcurrent,
		// "eject": getPrefixedFieldNameEjectRandom,
	}
	for _, numGoroutines := range []int{1, 50, 300} {
		for name, f := range funcs {
			for _, numFields := range []int{50, 500, 2000} {
				b.Run(fmt.Sprintf("%s-f%d-g%d", name, numFields, numGoroutines), func(b *testing.B) {
					names := make([]string, numFields)
					for i := 0; i < numFields; i++ {
						names[i] = getRandomString()
					}
					concurrentMap = cmap.New[string]()
					b.ResetTimer()
					wg := sync.WaitGroup{}
					count := b.N / numGoroutines
					if count == 0 {
						count = 1
					}
					for g := 0; g < numGoroutines; g++ {
						wg.Add(1)
						go func() {
							for n := 0; n < count; n++ {
								f(names[rand.Intn(numFields)])
							}
							wg.Done()
						}()
					}
					wg.Wait()
				})
			}
		}
	}
}
