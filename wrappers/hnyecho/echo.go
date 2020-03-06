package hnyecho

import (
	"sync"

	"github.com/honeycombio/beeline-go/wrappers/common"
	echo "github.com/labstack/echo/v4"
)

// EchoWrapper provides Honeycomb instrumentation for the Echo router via middleware
type (
	EchoWrapper struct {
		handlerNames map[string]string
		once         sync.Once
	}
)

// New returns a new EchoWrapper struct
func New() *EchoWrapper {
	return &EchoWrapper{}
}

// Middleware returns an echo.MiddlewareFunc to be used with Echo.Use()
func (e *EchoWrapper) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			r := c.Request()
			// get a new context with our trace from the request
			ctx, span := common.StartSpanOrTraceFromHTTP(r)
			defer span.Send()
			// push the context with our trace and span on to the request
			c.SetRequest(r.WithContext(ctx))

			// get name of handler
			handlerName := e.handlerName(c)
			if handlerName == "" {
				handlerName = "handler"
			}
			span.AddField("handler.name", handlerName)
			span.AddField("name", handlerName)

			// add route related fields
			span.AddField("route", c.Path())
			span.AddField("route.handler", handlerName)
			for _, name := range c.ParamNames() {
				// add field for each path param
				span.AddField("route.params."+name, c.Param(name))
			}

			// invoke next middleware in chain
			err := next(c)

			// add fields for http response code and size
			span.AddField("response.status_code", c.Response().Status)
			span.AddField("response.size", c.Response().Size)

			return err
		}
	}
}

// Unfortunately the name of c.Handler() is an anonymous function
// (https://github.com/labstack/echo/blob/master/echo.go#L487-L494).
// This function will return the correct handler name by building a
// map of request paths to actual handler names (only during the first
// request thus providing quick lookup for every request thereafter).
func (e *EchoWrapper) handlerName(c echo.Context) string {
	// only perform once
	e.once.Do(func() {
		// build map of request paths to handler names
		routes := c.Echo().Routes()
		e.handlerNames = make(map[string]string, len(routes))
		for _, r := range c.Echo().Routes() {
			e.handlerNames[r.Method+r.Path] = r.Name
		}
	})

	// lookup handler name for this request
	return e.handlerNames[c.Request().Method+c.Path()]
}
