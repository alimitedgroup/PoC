package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"

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

func getMerce(c *gin.Context) {
	id := c.Param("id")
	log.Printf("got /merce/%s request\n", id)
	intID, err := strconv.Atoi(id)
	if err != nil {
		log.Printf("invalid ID format: %s\n", id)
		c.JSON(http.StatusBadRequest, map[string]any{
			"error": fmt.Sprintf("invalid ID format: %s", id),
		})
		return
	}

	db := c.Value("db").(*sql.DB)
	if db == nil {
		log.Fatalf("Error getting db from context")
	}

	var merceName string
	var stockMerce int
	err = db.QueryRow("SELECT name, stock FROM merce WHERE id = ?", intID).Scan(&merceName, &stockMerce)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, map[string]any{
				"error": fmt.Sprintf("merce ID %s not found", id),
			})
			return
		}
		log.Printf("error querying db: %v\n", err)
		c.JSON(http.StatusInternalServerError, map[string]any{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, map[string]any{
		"merce": map[string]any{
			"id":   intID,
			"name": merceName,
		},
		"stock": stockMerce,
	})
}

func setupRoutes() *gin.Engine {
	r := gin.Default()
	r.GET("/", getRoot)
	r.GET("/health", getHealth)
	r.GET("/merce/:id", getMerce)

	return r
}

func startServer(listenPort string) {
	var r = setupRoutes()

	err := r.Run(fmt.Sprintf(":%v", listenPort))
	if err != nil {
		log.Fatal(err)
	}
}
