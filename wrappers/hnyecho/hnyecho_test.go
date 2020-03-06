package hnyecho

import (
	echo "github.com/labstack/echo/v4"
)

func ExampleMiddleware() {
	// assume you have handlers for hello and bye
	var hello echo.HandlerFunc
	var bye echo.HandlerFunc

	// set up Echo router with routes for hello and bye
	router := echo.New()
	router.GET("/hello/:name", hello)
	router.GET("/bye/:name", bye)

	// add hnyecho to middleware chain to provide honeycomb instrumentation
	router.Use(New().Middleware())
}
