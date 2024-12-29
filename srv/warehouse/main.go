package main

import (
	"context"
	"github.com/alimitedgroup/palestra_poc/common/messages"
	"github.com/nats-io/nats.go/jetstream"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/alimitedgroup/palestra_poc/common"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var meter = otel.Meter("github.com/alimitedgroup/palestra_poc/srv/warehouse")

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
	dbConnStr := os.Getenv("DB_URL")
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

	pg, err := pgxpool.New(ctx, dbConnStr)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to connect to PostgreSQL", "error", err)
		return
	}
	defer pg.Close()

	svc, js, err := setupWarehouse(ctx, nc, pg)
	if err != nil {
		return
	}
	defer func(svc *micro.Service) {
		err = (*svc).Stop()
		if err != nil {
			slog.ErrorContext(ctx, "Failed to gracefully stop service", "error", err)
		}
	}(&svc)

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

var warehouseId = "42"

func setupWarehouse(ctx context.Context, nc *nats.Conn, pg *pgxpool.Pool) (micro.Service, jetstream.JetStream, error) {
	// Create NATS microservice
	svc, err := micro.AddService(nc, micro.Config{Name: "catalog", Version: "0.0.1"})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to install service", "error", err)
		return nil, nil, err
	}
	grp := svc.AddGroup("warehouse")
	js, err := jetstream.New(nc)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to init JetStream", "error", err)
	}

	// Endpoint: `warehouse.ping`
	err = grp.AddEndpoint("ping", common.NewHandler(ctx, pg, PingHandler))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to add endpoint", "endpoint", "echo", "error", err)
		return nil, nil, err
	}

	// Endpoint: `stock_updates.<warehouseId>`
	err = InitStock(ctx, &js)
	if err != nil {
		return nil, nil, err
	}

	go startServer(8080)

	slog.InfoContext(ctx, "Service setup successful", "service", "warehouse", "warehouseId", warehouseId)

	return svc, js, nil
}
