package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/alimitedgroup/PoC/common"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go/jetstream"
)

func PingHandler(c *gin.Context) {
	c.JSON(200, gin.H{})
}

func CatalogHandler(s *common.Service[ApiGatewayState]) gin.HandlerFunc {
	return func(c *gin.Context) {
		state := s.State()
		watcher, err := state.catalogKV.WatchAll(c, jetstream.IgnoreDeletes())
		if err != nil {
			slog.ErrorContext(context.Background(), "Failed to watch kv bucket", "bucket", "catalog", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}

		out := make(map[string]string)
		for i := range watcher.Updates() {
			if i == nil {
				if err := watcher.Stop(); err != nil {
					slog.ErrorContext(context.Background(), "Failed to stop kv watcher", "bucket", "catalog", "error", err)
				}
				break
			}

			var row struct {
				Id   string `json:"id"`
				Name string `json:"name"`
			}
			err := json.Unmarshal(i.Value(), &row)
			if err != nil {
				slog.ErrorContext(context.Background(), "Failed to unmarshal kv payload", "bucket", "catalog", "error", err, "data", string(i.Value()))
				c.JSON(http.StatusInternalServerError, gin.H{"error": err})
				return
			}

			out[row.Id] = row.Name
		}

		c.JSON(http.StatusOK, out)
	}
}

func CatalogCreateHandler(s *common.Service[ApiGatewayState]) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		r, err := s.NatsConn().Request(
			"catalog.create",
			body,
			time.Second*2,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, map[string]any{"response": string(r.Data)})
	}
}
