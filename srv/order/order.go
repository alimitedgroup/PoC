package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"

	"github.com/alimitedgroup/PoC/common"
	"github.com/nats-io/nats.go"
)

type stockState struct {
	sync.Mutex
	// map of warehouse id to map of good id to amount of goods
	m map[string]map[string]int
}

type orderState struct {
	stock stockState
}

func setupObservability(ctx context.Context, otlpUrl string) func(context.Context) {
	otelshutdown := common.SetupOTelSDK(ctx, otlpUrl)

	return otelshutdown
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	natsUrl := os.Getenv("NATS_URL")
	otlpUrl := os.Getenv("OTLP_URL")

	otelshutdown := setupObservability(ctx, otlpUrl)
	defer otelshutdown(context.WithoutCancel(ctx))

	nc, err := nats.Connect(natsUrl)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to connect to NATS", "error", err)
		return
	}

	svc := common.NewService(ctx, nc, orderState{
		stock: stockState{sync.Mutex{}, make(map[string]map[string]int)},
	})

	if common.CreateStream(ctx, svc.JetStream(), common.StockUpdatesStreamConfig) != nil {
		slog.ErrorContext(ctx, "Failed to create stream", "stream", common.StockUpdatesStreamConfig.Name)
		return
	}

	svc.RegisterJsHandler(common.StockUpdatesStreamConfig.Name, StockUpdateHandler, common.WithSubjectFilter("stock_updates.>"))
	svc.RegisterHandler("order.ping", PingHandler)
	svc.RegisterHandler("order.create", CreateOrderHandler)

	// Wait for ctrl-c, and gracefully stop service
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	slog.InfoContext(ctx, "Shutting down")
	cancel()
}
