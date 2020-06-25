package hnygingonic

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/honeycombio/beeline-go"
)

func ExampleMiddleware() {
	// Setup a new gin Router, not using the Default here so that we can put the
	// Beeline middleware in before the middle provided by Gin
	router := gin.New()
	router.Use(
		Middleware(),
		gin.Logger(),
		gin.Recovery(),
	)

	// Setup the routes we want to use
	router.GET("/", home)
	router.GET("/alive", alive)
	router.GET("/ready", ready)

	// wrap the main router to set everything up for instrumenting
	log.Fatal(router.Run("127.0.0.1:8080"))
}

func home(c *gin.Context) {
	beeline.AddField(c, "Welcome", "Home")
	c.Data(http.StatusOK, "text/plain", []byte(`Welcome Home`))
}

func alive(c *gin.Context) {
	beeline.AddField(c, "Alive", true)
	c.Data(http.StatusOK, "text/plain", []byte(`OK`))
}

func ready(c *gin.Context) {
	beeline.AddField(c, "Ready", true)
	c.Data(http.StatusOK, "text/plain", []byte(`OK`))
}
