package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/alimitedgroup/PoC/common"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type catalogState struct {
	db *pgxpool.Pool
	kv jetstream.KeyValue
}

var meter = otel.Meter("github.com/alimitedgroup/PoC/srv/catalog")

func setupObservability(ctx context.Context, otlpUrl string) func(context.Context) {
	otelshutdown := common.SetupOTelSDK(ctx, otlpUrl)

	var err error
	common.NumRequests, err = meter.Int64Counter("num_requests", metric.WithDescription("Number of NATS requests received"))
	if err != nil {
		slog.ErrorContext(ctx, "failed to init metric `num_requests`", "error", err)
	}

	common.ResponseTime, err = meter.Int64Histogram("request_response_time", metric.WithDescription("Response time of request handlers in millisecond"))
	if err != nil {
		slog.ErrorContext(ctx, "failed to init `request_response_time`", "error", err)
	}

	return otelshutdown
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	dbConnStr := os.Getenv("DB_URL")
	natsUrl := os.Getenv("NATS_URL")
	otlpUrl := os.Getenv("OTLP_URL")

	otelshutdown := setupObservability(ctx, otlpUrl)
	defer otelshutdown(context.WithoutCancel(ctx))

	pool, err := pgxpool.New(ctx, dbConnStr)
	if err != nil {
		slog.ErrorContext(ctx, "failed to connect to database", "error", err)
		return
	}

	nc, err := nats.Connect(natsUrl)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to connect to NATS", "error", err)
		return
	}

	svc := common.NewService(ctx, nc, catalogState{db: pool})

	kv, err := svc.JetStream().CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:  "catalog",
		Storage: jetstream.FileStorage,
	})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create key-value store", "error", err)
		return
	}
	svc.State().kv = kv

	svc.RegisterHandler("catalog.ping", PingHandler)
	svc.RegisterHandler("catalog.create", CreateHandler)
	svc.RegisterHandler("catalog.list", ListHandler)
	svc.RegisterHandler("catalog.get", GetHandler)
	svc.RegisterHandler("catalog.update", UpdateHandler)
	svc.RegisterHandler("catalog.delete", DeleteHandler)

	// Wait for ctrl-c, and gracefully stop service
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	slog.InfoContext(ctx, "Shutting down")
	cancel()
}
