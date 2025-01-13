package main

import (
	"github.com/gin-gonic/gin"
	"log"
)

func main() {
	r := gin.Default()
	r.GET("/ping", PingHandler)
	err := r.Run(":8080")
	if err != nil {
		log.Fatal(err)
	}
}

func PingHandler(c *gin.Context) {
	c.JSON(200, gin.H{})
}
