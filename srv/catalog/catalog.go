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

var meter = otel.Meter("github.com/alimitedgroup/palestra_poc/srv/catalog")

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

	svc, err := setupCatalog(ctx, nc, pg)
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

func setupCatalog(ctx context.Context, nc *nats.Conn, pg *pgxpool.Pool) (micro.Service, error) {
	// Create NATS microservice
	svc, err := micro.AddService(nc, micro.Config{Name: "catalog", Version: "0.0.1"})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to install service", "error", err)
		return nil, err
	}
	grp := svc.AddGroup("catalog")

	// Endpoint: `catalog.ping`
	err = grp.AddEndpoint("ping", common.NewHandler(ctx, pg, PingHandler))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to add endpoint", "endpoint", "echo", "error", err)
		return nil, err
	}

	// Endpoint: `catalog.create`
	err = grp.AddEndpoint("create", common.NewHandler(ctx, pg, CreateHandler))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to add endpoint", "endpoint", "create", "error", err)
		return nil, err
	}

	// Endpoint: `catalog.list`
	err = grp.AddEndpoint("list", common.NewHandler(ctx, pg, ListHandler))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to add endpoint", "endpoint", "list", "error", err)
		return nil, err
	}

	// Endpoint: `catalog.get`
	err = grp.AddEndpoint("get", common.NewHandler(ctx, pg, GetHandler))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to add endpoint", "endpoint", "get", "error", err)
		return nil, err
	}

	// Endpoint: `catalog.update`
	err = grp.AddEndpoint("update", common.NewHandler(ctx, pg, UpdateHandler))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to add endpoint", "endpoint", "update", "error", err)
		return nil, err
	}

	// Endpoint: `catalog.delete`
	err = grp.AddEndpoint("delete", common.NewHandler(ctx, pg, DeleteHandler))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to add endpoint", "endpoint", "delete", "error", err)
		return nil, err
	}

	return svc, nil
}
