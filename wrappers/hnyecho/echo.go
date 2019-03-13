package hnyecho

import (
	"github.com/honeycombio/beeline-go/wrappers/common"
	"github.com/labstack/echo"
)

// EchoWrapper provides Honeycomb instrumentation for the Echo router via middleware
type (
	EchoWrapper struct {
		handlerNames map[string]string
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

			// add field for each path param
			for _, name := range c.ParamNames() {
				span.AddField("app."+name, c.Param(name))
			}

			// add field for each query param
			for name, values := range c.QueryParams() {
				if len(values) == 1 {
					span.AddField("app."+name, values[0])
				} else {
					span.AddField("app."+name, values)
				}
			}

			// invoke next middleware in chain
			err := next(c)

			// add field for http response code
			span.AddField("response.status_code", c.Response().Status)

			return err
		}
	}
}

// Unfortunately the name of c.Handler() is an anonymous function
// (https://github.com/labstack/echo/blob/master/echo.go#L487-L494).
// This function will build a map of request paths to actual handler names during
// the first request, thus providing quick lookup for every request thereafter.
func (e *EchoWrapper) handlerName(c echo.Context) string {
	// if first request
	if e.handlerNames == nil {
		// build map of request paths to handler names
		routes := c.Echo().Routes()
		e.handlerNames = make(map[string]string, len(routes))
		for _, r := range c.Echo().Routes() {
			e.handlerNames[r.Method+r.Path] = r.Name
		}
	}

	// lookup handler name for this request
	return e.handlerNames[c.Request().Method+c.Path()]
}
