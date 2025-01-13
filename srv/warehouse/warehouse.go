package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"

	"github.com/alimitedgroup/PoC/common"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type warehouseState struct {
	stock       stockState
	reservation reservationState
}

var meter = otel.Meter("github.com/alimitedgroup/PoC/srv/warehouse")
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

	srv := common.NewService(ctx, nc, warehouseState{
		stock:       stockState{sync.Mutex{}, make(map[uint64]int), make(map[uint64]int)},
		reservation: reservationState{sync.Mutex{}, make([]Reservation, 0)},
	})

	err = InitWarehouse(ctx, srv)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to initialize the service", "error", err)
		return
	}

	srv.RegisterHandler(fmt.Sprintf("warehouse.ping.%s", warehouseId), PingHandler)
	srv.RegisterHandler(fmt.Sprintf("warehouse.add_stock.%s", warehouseId), AddStockHandler)
	srv.RegisterHandler(fmt.Sprintf("warehouse.reserve.%s", warehouseId), ReserveHandler)

	slog.InfoContext(ctx, "Service setup successful", "service", "warehouse", "warehouseId", warehouseId)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	cancel()

	slog.InfoContext(ctx, "Shutting down")
}

func InitWarehouse(ctx context.Context, srv *common.Service[warehouseState]) error {
	err := common.CreateStream(ctx, srv.JetStream(), common.StockUpdatesStreamConfig)
	if err != nil {
		return fmt.Errorf("failed to create stock_updates stream: %w", err)
	}

	srv.RegisterJsHandlerExisting(common.StockUpdatesStreamConfig.Name, StockUpdateHandler, common.WithSubjectFilter("stock_updates.>"))
	slog.InfoContext(ctx, "Stock updates handled", "stock", srv.State().stock.r)

	err = common.CreateStream(ctx, srv.JetStream(), common.ReservationStreamConfig)
	if err != nil {
		return fmt.Errorf("failed to create reservations stream: %w", err)
	}

	srv.RegisterJsHandlerExisting(common.ReservationStreamConfig.Name, ReservationHandler, common.WithSubjectFilter("reservations.>"))
	slog.InfoContext(ctx, "Reservations handled", "reservation", srv.State().reservation.s)
	go removeReservationsLoop(ctx, &srv.State().reservation)

	return nil
}
