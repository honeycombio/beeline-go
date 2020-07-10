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
		// Add the beeline middleware to the chain
		Middleware(nil),
		// Doing something like the following would have the Middleware grab specifc
		// GET query params that you deal with in your gin application.
		//Middleware(map[string]struct{}{
		//"parts":  {},
		//"limit":  {},
		//"offset": {},
		//})
		// The Logger and Recovery middleware which are setup in the Default gin router
		gin.Logger(),
		gin.Recovery(),
		// Our example middleware that does extra work
		exampleMiddleware(),
	)

	// Setup the routes we want to use
	router.GET("/", home)
	router.GET("/alive", alive)
	router.GET("/ready", ready)

	// Start the server
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
	c.Data(http.StatusOK, "text/plain", []byte(`OK`))
}

func ready(c *gin.Context) {
	_, span := StartSpan(c, "main.ready")
	defer span.Send()
	// Do some work here
	span.AddField("Ready", true)
	c.Data(http.StatusOK, "text/plain", []byte(`OK`))
}

func exampleMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		hnyctx, span := StartSpan(c, "main.exampleMiddleware")
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
