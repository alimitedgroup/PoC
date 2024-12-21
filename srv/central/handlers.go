package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	db *sql.DB
}

func getRoot(c *gin.Context) {
	log.Printf("got / request\n")
	c.String(http.StatusOK, "Hello!\n")
}

func getHealth(c *gin.Context) {
	log.Printf("got /health request\n")
	c.String(http.StatusOK, "OK\n")
}

func (r *Controller) getMerce(c *gin.Context) {
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

	var merceName string
	var stockMerce int
	err = r.db.QueryRow("SELECT name, stock FROM merce WHERE id = $1", intID).Scan(&merceName, &stockMerce)
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

func setupRoutes(db *sql.DB) *gin.Engine {
	controller := Controller{
		db: db,
	}

	r := gin.Default()
	r.GET("/", getRoot)
	r.GET("/health", getHealth)
	r.GET("/merce/:id", controller.getMerce)

	return r
}

func startServer(listenPort string, db *sql.DB) {
	var r = setupRoutes(db)

	err := r.Run(fmt.Sprintf(":%v", listenPort))
	if err != nil {
		log.Fatal(err)
	}
}
