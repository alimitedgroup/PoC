package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func getRoot(c *gin.Context) {
	log.Printf("got / request\n")
	c.String(http.StatusOK, "Hello!\n")
}

func getHealth(c *gin.Context) {
	log.Printf("got /health request\n")
	c.String(http.StatusOK, "OK\n")
}

func setupRoutes() *gin.Engine {
	r := gin.Default()
	r.GET("/", getRoot)
	r.GET("/health", getHealth)

	return r
}

func startServer(listenPort int) {
	var r = setupRoutes()

	err := r.Run(fmt.Sprintf(":%d", listenPort))
	if err != nil {
		log.Fatal(err)
	}
}
