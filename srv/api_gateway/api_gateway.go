package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/alimitedgroup/PoC/common"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/puzpuzpuz/xsync/v3"
)

type ApiGatewayState struct {
	stock *xsync.MapOf[string, *xsync.MapOf[string, int]]
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

	svc := common.NewService(ctx, nc, ApiGatewayState{stock: xsync.NewMapOf[string, *xsync.MapOf[string, int]]()})
	svc.RegisterJsHandler("stock_updates", StockUpdateHandler)

	r := gin.Default()
	r.GET("/ping", PingHandler)
	r.GET("/warehouses", WarehouseListRoute(svc))
	r.GET("/stock/:warehouseId", StockGetRoute(svc))
	err = r.Run(":8080")
	if err != nil {
		log.Fatal(err)
	}
}

func PingHandler(c *gin.Context) {
	c.JSON(200, gin.H{})
}
