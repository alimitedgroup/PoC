package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/alimitedgroup/palestra_poc/common"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var meter = otel.Meter("github.com/alimitedgroup/palestra_poc/srv/warehouse")
var warehouseId = os.Getenv("WAREHOUSE_ID")
var js jetstream.JetStream

func setupObservability(ctx context.Context, otlpUrl string) func(context.Context) {
	otelshutdown := common.SetupOTelSDK(ctx, otlpUrl)

	var err error
	common.NumRequests, err = meter.Int64Counter("num_requests", metric.WithDescription("Number of NATS requests received"))
	if err != nil {
		slog.ErrorContext(ctx, "failed to init metric `num_requests`", "error", err)
	}

	return otelshutdown
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	natsUrl := os.Getenv("NATS_URL")
	otlpUrl := os.Getenv("OTLP_URL")

	otelshutdown := setupObservability(ctx, otlpUrl)
	defer func() {
		otelshutdown(context.WithoutCancel(ctx))
	}()

	nc, err := nats.Connect(natsUrl)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to connect to NATS", "error", err)
		return
	}
	defer nc.Close()

	js, err = jetstream.New(nc)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to init JetStream", "error", err)
		return
	}

	err = setupWarehouse(ctx, nc, js)
	if err != nil {
		return
	}

	// Wait for ctrl-c, and gracefully stop service
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	cancel()

	slog.InfoContext(ctx, "Shutting down")
}

func setupWarehouse(ctx context.Context, nc *nats.Conn, js jetstream.JetStream) error {
	// TODO: cleanly unsubscribe on ctrl-c

	// Initialization
	err := InitStock(ctx, js)
	if err != nil {
		return err
	}
	err = InitReservations(ctx, js)
	if err != nil {
		return err
	}

	// Endpoint: `stock_updates.<warehouseId>`
	_, err = nc.Subscribe(fmt.Sprintf("stock_updates.%s", warehouseId), func(req *nats.Msg) {
		StockUpdateHandler(ctx, req)
	})
	if err != nil {
		slog.ErrorContext(
			ctx, "Failed to subscribe to NATS",
			"error", err,
			"subject", fmt.Sprintf("stock_updates.%s", warehouseId),
		)
		return err
	}

	// Endpoint: `warehouse.reserve.<warehouseId>`
	_, err = nc.Subscribe(fmt.Sprintf("warehouse.reserve.%s", warehouseId), func(req *nats.Msg) {
		ReserveHandler(ctx, req)
	})
	if err != nil {
		slog.ErrorContext(
			ctx, "Failed to subscribe to NATS",
			"error", err,
			"subject", fmt.Sprintf("warehouse.reserve.%s", warehouseId),
		)
		return err
	}

	// Endpoint: `warehouse.ping`
	_, err = nc.Subscribe(fmt.Sprintf("warehouse.ping.%s", warehouseId), PingHandler)
	if err != nil {
		slog.ErrorContext(
			ctx, "Failed to subscribe to NATS",
			"error", err,
			"subject", fmt.Sprintf("warehouse.ping.%s", warehouseId),
		)
		return err
	}

	slog.InfoContext(ctx, "Service setup successful", "service", "warehouse", "warehouseId", warehouseId)

	return nil
}
