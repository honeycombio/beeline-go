package hnygrpc

import (
	"context"
	"testing"

	beeline "github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/wrappers/config"
	libhoney "github.com/honeycombio/libhoney-go"
	"github.com/honeycombio/libhoney-go/transmission"
	"github.com/stretchr/testify/assert"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

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
		"content-type": "application/grpc",
		":authority":   "api.honeycomb.io:443",
		"user-agent":   "testing-is-fun",
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

	method, ok := successfulFields["handler.method"]
	assert.True(t, ok, "method name should be set")
	assert.Equal(t, "test.method", method, "method name should be set")

	status, ok := successfulFields["response.grpc_status_code"]
	assert.True(t, ok, "Status code must exist on middleware generated event")
	assert.Equal(t, codes.OK, status, "status must exist")
}
