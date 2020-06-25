package hnygingonic

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/trace"
	"github.com/honeycombio/beeline-go/wrappers/common"
)

// Middleware wraps httprouter handlers. Since it wraps handlers with explicit
// parameters, it can add those values to the event it generates.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// get a new context with our trace from the request, and add common fields
		ctx, span := common.StartSpanOrTraceFromHTTP(c.Request)
		defer span.Send()
		// Add the span context to the gin context as we need to be able to pass
		// this context around our gin application
		c.Set("beeline-middleware-context", ctx)
		// push the context with our trace and span on to the request
		c.Request = c.Request.WithContext(ctx)

		// replace the writer with our wrapper to catch the status code
		wrappedWriter := common.NewResponseWriter(c.Writer)

		// pull out any variables in the URL, add the thing we're matching, etc.
		for _, param := range c.Params {
			span.AddField("handler.vars."+param.Key, param.Value)
		}

		// pull out any GET query params
		for key, value := range c.Request.URL.Query() {
			if len(value) > 1 {
				span.AddField("handler.query."+key, value)
			} else if len(value) == 1 {
				span.AddField("handler.query."+key, value[0])
			} else {
				span.AddField("handler.query."+key, nil)
			}
		}

		name := c.HandlerName()
		span.AddField("handler.name", name)
		span.AddField("name", name)
		// Run the next function in the Middleware chain
		c.Next()

		if wrappedWriter.Status == 0 {
			wrappedWriter.Status = 200
		}

		span.AddField("response.status_code", wrappedWriter.Status)
	}
}

// StartSpan is a helper function to start a new span in a gin-gonic context
// This is required because the gin-gonic handler function expects to receive
// *gin.Context rather than context.Context
func StartSpan(c *gin.Context, name string) (context.Context, *trace.Span) {
	beelineContext, exists := c.Get("beeline-middleware-context")
	var ctx context.Context

	if exists {
		ctx, _ = beelineContext.(context.Context)
	}

	return beeline.StartSpan(ctx, name)
}

// SetContext should be used to replace the context.Context in the gin.Context
// in the case of having multiple custom middleware in the codebase
func SetContext(c *gin.Context, newMiddleWareContext context.Context) {
	c.Set("beeline-middleware-context", newMiddleWareContext)
}
