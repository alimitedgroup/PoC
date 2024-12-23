package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

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
	ctx := context.Background()
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

	svc, err := setupWarehouse(ctx, nc, pg)
	if err != nil {
		return
	}
	defer func(svc *micro.Service) {
		err = (*svc).Stop()
		if err != nil {
			slog.ErrorContext(ctx, "Failed to gracefully stop service", "error", err)
		}
	}(&svc)

	// Wait for ctrl-c, and gracefully stop service
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	slog.InfoContext(ctx, "Shutting down")
}

func setupWarehouse(ctx context.Context, nc *nats.Conn, pg *pgxpool.Pool) (micro.Service, error) {
	// Create NATS microservice
	svc, err := micro.AddService(nc, micro.Config{Name: "catalog", Version: "0.0.1"})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to install service", "error", err)
		return nil, err
	}
	grp := svc.AddGroup("warehouse")

	// Endpoint: `warehouse.ping`
	err = grp.AddEndpoint("ping", common.NewHandler(ctx, pg, PingHandler))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to add endpoint", "endpoint", "echo", "error", err)
		return nil, err
	}

	go startServer(":8080")

	return svc, nil
}
