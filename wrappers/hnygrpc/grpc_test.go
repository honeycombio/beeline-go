package hnygrpc

import (
	"context"
	"testing"

	beeline "github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/propagation"
	"github.com/honeycombio/beeline-go/trace"
	"github.com/honeycombio/beeline-go/wrappers/config"
	libhoney "github.com/honeycombio/libhoney-go"
	"github.com/honeycombio/libhoney-go/transmission"
	"github.com/stretchr/testify/assert"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func TestStartSpanOrTrace(t *testing.T) {
	info := &grpc.UnaryServerInfo{
		FullMethod: "test.method",
	}
	// no current span, no parser hook, expect a new trace
	ctx := context.Background()
	ctx, span := startSpanOrTraceFromUnaryGRPC(ctx, info, nil)
	assert.Equal(t, 0, len(span.GetChildren()), "Span should not have children")
	assert.Equal(t, "", span.GetParentID(), "Span should not have parent")

	// now let's create a child span
	ctx = trace.PutSpanInContext(ctx, span)
	ctx, spanTwo := startSpanOrTraceFromUnaryGRPC(ctx, info, nil)
	assert.Equal(t, 1, len(span.GetChildren()), "Should have one child span")
	assert.Equal(t, span, spanTwo.GetParent(), "Span should have been created as child")

	// metadata, no parser hook
	ctx = context.Background()
	ctx = metadata.NewIncomingContext(ctx, metadata.New(map[string]string{
		"content-type": "application/grpc",
	}))
	ctx, spanThree := startSpanOrTraceFromUnaryGRPC(ctx, info, nil)
	assert.Equal(t, 0, len(spanThree.GetChildren()), "span should not have children")
	assert.Equal(t, "", span.GetParentID(), "Span should not have parent")

	// metadata, no parser hook, x-honeycomb-trace header
	ctx = metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
		"x-honeycomb-trace": "1;trace_id=4bf92f3577b34da6a3ce929d0e0e473,parent_id=00f067aa0ba902b7,context=",
	}))
	ctx, spanFour := startSpanOrTraceFromUnaryGRPC(ctx, info, nil)
	assert.Equal(t, 0, len(spanFour.GetChildren()), "span should not have children")
	assert.Equal(t, "00f067aa0ba902b7", spanFour.GetParentID(), "Expected parent_id from header")
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e473", spanFour.GetTrace().GetTraceID(), "Expected trace id from header")

	// metadata, parserhook
	ctx = metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
		"content-type": "application/grpc",
	}))
	parserHook := func(ctx context.Context) *propagation.PropagationContext {
		return &propagation.PropagationContext{
			TraceID:  "fffffffffffffffffffffffffffffff",
			ParentID: "aaaaaaaaaaaaaaaa",
		}
	}
	ctx, spanFive := startSpanOrTraceFromUnaryGRPC(ctx, info, parserHook)
	assert.Equal(t, 0, len(spanFive.GetChildren()), "span should not have children")
	assert.Equal(t, "aaaaaaaaaaaaaaaa", spanFive.GetParentID(), "Expected parent id from propagation context")
	assert.Equal(t, "fffffffffffffffffffffffffffffff", spanFive.GetTrace().GetTraceID(), "Expected trace id from propagation context")
}

func TestUnaryInterceptor(t *testing.T) {
	mo := &transmission.MockSender{}
	client, err := libhoney.NewClient(libhoney.ClientConfig{
		APIKey:       "placeholder",
		Dataset:      "placeholder",
		APIHost:      "placeholder",
		Transmission: mo})
	assert.Equal(t, nil, err)
	beeline.Init(beeline.Config{Client: client})

	md := metadata.New(map[string]string{
		"content-type":      "application/grpc",
		":authority":        "api.honeycomb.io:443",
		"user-agent":        "testing-is-fun",
		"X-Forwarded-For":   "10.11.12.13", // headers are Kabob-Title-Case from clients
		"X-Forwarded-Proto": "https",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return req, nil
	}
	info := &grpc.UnaryServerInfo{
		FullMethod: "test.method",
	}
	interceptor := UnaryServerInterceptorWithConfig(config.GRPCIncomingConfig{})
	var dummy interface{}
	resp, err := interceptor(ctx, dummy, info, handler)
	assert.NoError(t, err, "Unexpected error calling interceptor")
	assert.Equal(t, resp, dummy)

	evs := mo.Events()
	assert.Equal(t, 1, len(evs), "1 event is created")
	successfulFields := evs[0].Data

	contentType, ok := successfulFields["request.content_type"]
	assert.True(t, ok, "content-type field must exist on middleware generated event")
	assert.Equal(t, "application/grpc", contentType, "content-type should be set")

	authority, ok := successfulFields["request.header.authority"]
	assert.True(t, ok, "authority field must exist on middleware generated event")
	assert.Equal(t, "api.honeycomb.io:443", authority, "authority should be set")

	userAgent, ok := successfulFields["request.header.user_agent"]
	assert.True(t, ok, "user-agent expected to exist on middleware generated event")
	assert.Equal(t, "testing-is-fun", userAgent, "user-agent should be set")

	xForwardedFor, ok := successfulFields["request.header.x_forwarded_for"]
	assert.True(t, ok, "x_forwarded_for expected to exist on middleware generated event")
	assert.Equal(t, "10.11.12.13", xForwardedFor, "x_forwarded_for should be set")

	xForwardedProto, ok := successfulFields["request.header.x_forwarded_proto"]
	assert.True(t, ok, "x_forwarded_proto expected to exist on middleware generated event")
	assert.Equal(t, "https", xForwardedProto, "x_forwarded_proto should be set")

	method, ok := successfulFields["handler.method"]
	assert.True(t, ok, "method name should be set")
	assert.Equal(t, "test.method", method, "method name should be set")

	status, ok := successfulFields["response.grpc_status_code"]
	assert.True(t, ok, "Status code must exist on middleware generated event")
	assert.Equal(t, codes.OK, status, "status must exist")

	statusMsg, ok := successfulFields["response.grpc_status_message"]
	assert.True(t, ok, "Status message must exist on middleware generated event")
	assert.Equal(t, codes.OK.String(), statusMsg, "human-readable status must exist")
}
