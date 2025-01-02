package main

import (
	"context"
	"fmt"
	"github.com/alimitedgroup/palestra_poc/common/messages"
	"github.com/nats-io/nats.go/jetstream"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/alimitedgroup/palestra_poc/common"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var meter = otel.Meter("github.com/alimitedgroup/palestra_poc/srv/warehouse")
var warehouseId = os.Getenv("WAREHOUSE_ID")

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

	js, err := jetstream.New(nc)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to init JetStream", "error", err)
		return
	}

	err = setupWarehouse(ctx, nc, js)
	if err != nil {
		return
	}

	// Random testing loop
	go func(ctx context.Context) {
		for {
			if ctx.Err() != nil {
				break
			}

			time.Sleep(200 * time.Millisecond)
			stock.Lock()

			if _, ok := stock.s[10]; !ok {
				stock.s[10] = 0
			}

			err := SendStockUpdate(ctx, &js, &messages.StockUpdate{
				{GoodId: 10, Amount: stock.s[10] + 2},
			})
			stock.Unlock()
			if err != nil {
				slog.ErrorContext(ctx, "Failed to send stock update", "error", err)
			}

			slog.DebugContext(ctx, "Sent stock update")
		}
	}(ctx)

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
	// Enpoint: `warehouse.reserve_goods.<warehouseId>`

	// Endpoint: `warehouse.reserve.<warehouseId>`
	_, err = nc.Subscribe(warehouseId, func(msg *nats.Msg) {})

	// Endpoint: `warehouse.ping`
	_, err = nc.Subscribe(fmt.Sprintf("warehouse.ping.%s", warehouseId), func(msg *nats.Msg) {
		_ = msg.Respond([]byte("pong"))
	})
	if err != nil {
		slog.ErrorContext(
			ctx, "Failed to subscribe to nats",
			"error", err,
			"subject", fmt.Sprintf("warehouse.ping.%s", warehouseId),
		)
		return err
	}

	slog.InfoContext(ctx, "Service setup successful", "service", "warehouse", "warehouseId", warehouseId)

	return nil
}
