package hnygingonic

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/honeycombio/beeline-go"
)

func ExampleMiddleware() {
	// Setup a new gin Router, not using the Default here so that we can put the
	// Beeline middleware in before the middleware provided by Gin
	router := gin.New()
	router.Use(
		Middleware(nil),
		// Doing something like the following would have the Middleware grab specifc
		// GET query params that you deal with in your gin application.
		//Middleware(map[string]struct{}{
		//"parts":  {},
		//"limit":  {},
		//"offset": {},
		//})
		gin.Logger(),
		gin.Recovery(),
		exampleWrapper(),
	)

	// Setup the routes we want to use
	router.GET("/", home)
	router.GET("/alive", alive)
	router.GET("/ready", ready)

	// wrap the main router to set everything up for instrumenting
	log.Fatal(router.Run("127.0.0.1:8080"))
}

func home(c *gin.Context) {
	hnyctx, span := StartSpan(c, "main.home")
	defer span.Send()
	span.AddField("Welcome", "Home")
	childFunction(hnyctx)
	c.Data(http.StatusOK, "text/plain", []byte(`Welcome Home`))
}

func alive(c *gin.Context) {
	_, span := StartSpan(c, "main.alive")
	defer span.Send()
	span.AddField("Alive", true)
	c.Data(http.StatusOK, "text/plain", []byte(`OK`))
}

func ready(c *gin.Context) {
	_, span := StartSpan(c, "main.ready")
	defer span.Send()
	span.AddField("Ready", true)
	c.Data(http.StatusOK, "text/plain", []byte(`OK`))
}

func exampleWrapper() gin.HandlerFunc {
	return func(c *gin.Context) {
		hnyctx, span := StartSpan(c, "main.exampleWrapper")
		defer span.Send()
		SetContext(c, hnyctx)
		// Do some work
		c.Next()
		childFunction(hnyctx)
	}
}

func childFunction(ctx context.Context) {
	_, span := beeline.StartSpan(ctx, "main.childFunction")
	defer span.Send()
	// Do some work here
}
