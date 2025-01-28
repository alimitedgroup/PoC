package main

import (
	"context"
	"encoding/json"
	"github.com/nats-io/nats.go/jetstream"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/alimitedgroup/PoC/common"
	"github.com/alimitedgroup/PoC/common/messages"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/puzpuzpuz/xsync/v3"
)

type ApiGatewayState struct {
	stock  *xsync.MapOf[string, *xsync.MapOf[string, int]]
	orders *xsync.MapOf[string, messages.OrderCreated]
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	natsUrl := os.Getenv("NATS_URL")
	defer cancel()

	nc, err := nats.Connect(natsUrl)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to connect to NATS", "error", err)
		return
	}

	svc := common.NewService(ctx, nc, ApiGatewayState{
		stock:  xsync.NewMapOf[string, *xsync.MapOf[string, int]](),
		orders: xsync.NewMapOf[string, messages.OrderCreated](),
	})

	if common.CreateStream(ctx, svc.JetStream(), common.StockUpdatesStreamConfig) != nil {
		slog.ErrorContext(ctx, "Failed to create stream", "stream", common.StockUpdatesStreamConfig.Name)
		return
	}
	if common.CreateStream(ctx, svc.JetStream(), common.OrdersStreamConfig) != nil {
		slog.ErrorContext(ctx, "Failed to create stream", "stream", common.OrdersStreamConfig.Name)
		return
	}
	svc.RegisterJsHandler("stock_updates", StockUpdateHandler)
	svc.RegisterJsHandler("orders", OrderCreateHandler)

	r := gin.Default()
	r.GET("/ping", PingHandler)
	r.GET("/catalog", CatalogHandler(svc))
	r.GET("/warehouses", WarehouseListRoute(svc))
	r.GET("/stock/:warehouseId", StockGetRoute(svc))
	r.POST("/stock/:warehouseId", StockPostRoute(svc))
	r.GET("/orders", OrderListRoute(svc))
	r.GET("/orders/:orderId", OrderGetRoute(svc))
	r.POST("/orders", OrderPostRoute(svc))
	err = r.Run(":8080")
	if err != nil {
		log.Fatal(err)
	}
}

func PingHandler(c *gin.Context) {
	c.JSON(200, gin.H{})
}

func CatalogHandler(s *common.Service[ApiGatewayState]) gin.HandlerFunc {
	kv, err := s.JetStream().CreateKeyValue(context.Background(), jetstream.KeyValueConfig{Bucket: "catalog"})
	if err != nil {
		slog.ErrorContext(context.Background(), "Failed to create kv bucket", "bucket", "catalog", "error", err)
		log.Fatal(err)
	}

	return func(c *gin.Context) {
		watcher, err := kv.Watch(c, "*")
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
