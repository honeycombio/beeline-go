package hnygrpc

import (
	"context"
	"reflect"
	"runtime"

	"github.com/honeycombio/beeline-go/propagation"
	"github.com/honeycombio/beeline-go/trace"
	"github.com/honeycombio/beeline-go/wrappers/config"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// getMetadataStringValue is a simpler helper method that checks the provided
// metadata for a value associated with the provided key. If the value exists,
// it is returned. If the value does not exist, an empty string is returned.
func getMetadataStringValue(md metadata.MD, key string) string {
	if val, ok := md[key]; ok {
		if len(val) > 0 {
			return val[0]
		}
		return ""
	}
	return ""
}

// startSpanOrTraceFromUnaryGRPC checks to see if a trace already exists in the
// provided context before creating either a root span or a child span of the
// existing active span. The function understands trace parser hooks, so if one
// is provided, it'll use it to parse the incoming request for trace context.
func startSpanOrTraceFromUnaryGRPC(
	ctx context.Context,
	info *grpc.UnaryServerInfo,
	parserHook config.GRPCTraceParserHook,
) (context.Context, *trace.Span) {
	span := trace.GetSpanFromContext(ctx)
	if span == nil {
		// no active span, create a new trace
		var tr *trace.Trace
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			if parserHook == nil {
				beelineHeader := getMetadataStringValue(md, propagation.TracePropagationGRPCHeader)
				ctx, tr = trace.NewTrace(ctx, beelineHeader)
			} else {
				prop := parserHook(ctx)
				ctx, tr = trace.NewTraceFromPropagationContext(ctx, prop)
			}
		} else {
			ctx, tr = trace.NewTrace(ctx, "")
		}
		span = tr.GetRootSpan()
	} else {
		// create new span as child of active span.
		ctx, span = span.CreateChild(ctx)
	}
	return ctx, span
}

// addFields just adds available information about a gRPC request to the provided span.
func addFields(ctx context.Context, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler, span *trace.Span) {
	handlerName := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()

	span.AddField("handler.name", handlerName)
	span.AddField("name", handlerName)
	span.AddField("handler.method", info.FullMethod)

	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if val, ok := md["content-type"]; ok {
			span.AddField("request.content_type", val[0])
		}
		if val, ok := md[":authority"]; ok {
			span.AddField("request.header.authority", val[0])
		}
		if val, ok := md["user-agent"]; ok {
			span.AddField("request.header.user_agent", val[0])
		}
	}
}

// UnaryServerInterceptorWithConfig will create a Honeycomb event per invocation of the
// returned interceptor. If passed a config.GRPCIncomingConfig with a GRPCParserHook,
// the hook will be called when creating the event, allowing it to specify how trace context
// information should be included in the span (e.g. it may have come from a remote parent in
// a specific format).
//
// Events created from GRPC interceptors will contain information from the gRPC metadata, if
// it exists, as well as information about the handler used and method being called.
func UnaryServerInterceptorWithConfig(cfg config.GRPCIncomingConfig) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		ctx, span := startSpanOrTraceFromUnaryGRPC(ctx, info, cfg.GRPCParserHook)
		defer span.Send()

		addFields(ctx, info, handler, span)
		resp, err := handler(ctx, req)
		if err != nil {
			span.AddTraceField("handler_error", err.Error())
		}
		span.AddField("response.grpc_status_code", status.Code(err))
		return resp, err
	}
}
