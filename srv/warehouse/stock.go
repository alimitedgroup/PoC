package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/alimitedgroup/PoC/common/messages"
	"github.com/alimitedgroup/PoC/common/natsutil"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// stock contains the currently stocked items inside of the field `s`,
// and the amounts of items that have been reserved, inside the `r` field.
// Each of these fields is a map from good id to stocked (or reserved) amount.
//
// Please note that this singleton should be locked before being used by
// calling its `Lock()` method.
var stock = struct {
	sync.Mutex
	s map[uint64]int
	r map[uint64]int
}{sync.Mutex{}, make(map[uint64]int), make(map[uint64]int)}

func InitStock(ctx context.Context, js jetstream.JetStream) error {
	stock.Lock()
	defer stock.Unlock()

	_, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     "stock_updates",
		Subjects: []string{"stock_updates.>"},
		Storage:  jetstream.FileStorage,
	})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create stream", "error", err, "stream", "stock_updates")
		return fmt.Errorf("failed to create stream: %w", err)
	}

	err = natsutil.ConsumeAll(
		ctx,
		js,
		"stock_updates",
		jetstream.OrderedConsumerConfig{FilterSubjects: []string{fmt.Sprintf("stock_updates.%s", warehouseId)}},
		func(msg jetstream.Msg) { InitStockUpdateHandler(ctx, msg); _ = msg.Ack() },
	)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to consume stock updates", "error", err, "stream", "stock_updates")
		return fmt.Errorf("failed to consume stock updates: %w", err)
	}

	slog.InfoContext(ctx, "Stock updates handled", "stock", stock.s)
	return nil
}

func InitStockUpdateHandler(ctx context.Context, req jetstream.Msg) {
	var msg messages.StockUpdate
	err := json.Unmarshal(req.Data(), &msg)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"Error unmarshalling message",
			"error", err,
			"subject", req.Subject(),
			"message", req.Headers()["Nats-Msg-Id"][0],
		)
		return
	}

	for _, row := range msg {
		// stock MUST be locked
		stock.s[row.GoodId] = row.Amount
		stock.r[row.GoodId] = 0
	}
}

func SendStockUpdate(ctx context.Context, js *jetstream.JetStream, msg *messages.StockUpdate) error {
	body, err := json.Marshal(msg)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to marshal value as JSON", "error", err)
		return err
	}

	_, err = (*js).PublishMsg(ctx, &nats.Msg{
		Subject: fmt.Sprintf("stock_updates.%s", warehouseId),
		Data:    body,
	})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to publish message", "error", err)
		return err
	}

	for _, row := range *msg {
		// stock MUST be locked
		stock.s[row.GoodId] = row.Amount
	}

	return nil
}
