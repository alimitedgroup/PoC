package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/alimitedgroup/PoC/common"
	"github.com/alimitedgroup/PoC/common/messages"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/puzpuzpuz/xsync/v3"
)

type ApiGatewayState struct {
	stock     *xsync.MapOf[string, *xsync.MapOf[string, int]]
	orders    *xsync.MapOf[string, messages.OrderCreated]
	catalogKV jetstream.KeyValue
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

	kv, err := svc.JetStream().CreateOrUpdateKeyValue(ctx, common.CatalogKeyValueConfig)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create key-value store", "error", err)
		return
	}
	svc.State().catalogKV = kv

	svc.RegisterJsHandler("stock_updates", StockUpdateHandler)
	svc.RegisterJsHandler("orders", OrderCreateHandler)

	r := gin.Default()
	r.GET("/ping", PingHandler)
	r.GET("/catalog", CatalogHandler(svc))
	r.POST("/catalog", CatalogCreateHandler(svc))
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
