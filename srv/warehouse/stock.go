package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/alimitedgroup/palestra_poc/common/messages"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"log/slog"
	"sync"
	"time"
)

var stock = struct {
	sync.Mutex
	s map[uint64]int
}{sync.Mutex{}, make(map[uint64]int)}

func InitStock(ctx context.Context, jstream *jetstream.JetStream) error {
	js := *jstream
	stock.Lock()
	defer stock.Unlock()

	_, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     "stock_updates",
		Subjects: []string{"stock_updates.>"},
		Storage:  jetstream.FileStorage,
	})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create stream", "error", err, "stream", "stock_updates")
		return err
	}

	// `OrderedConsumer`s are ephemeral, and so don't need to be deleted
	consumer, err := js.OrderedConsumer(ctx, "stock_updates", jetstream.OrderedConsumerConfig{
		FilterSubjects: []string{fmt.Sprintf("stock_updates.%s", warehouseId)},
	})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create consumer", "error", err, "stream", "stock_updates")
		return err
	}

	// Start consuming all messages
	sub, err := consumer.Consume(func(msg jetstream.Msg) {
		StockUpdateHandler(ctx, &msg)
		_ = msg.Ack()
	})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to consume messages", "error", err, "stream", "stock_updates")
		return err
	}

	// Loop to check if we consumed all messages
	for {
		info, err := consumer.Info(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to get consumer info", "error", err, "stream", "stock_updates")
			return err
		}

		if (*info).NumPending == 0 {
			sub.Stop()
			break
		}

		time.Sleep(250 * time.Millisecond)
	}

	slog.InfoContext(ctx, "Stock updates handled", "stock", stock.s)

	return nil
}

func StockUpdateHandler(ctx context.Context, req *jetstream.Msg) {
	var msg messages.StockUpdate
	err := json.Unmarshal((*req).Data(), &msg)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"Error unmarshalling message",
			"error", err,
			"subject", (*req).Subject,
			"message", (*req).Headers()["Nats-Msg-Id"][0],
		)
		return
	}

	for _, row := range msg {
		// stock MUST be locked
		stock.s[row.GoodId] = row.Amount
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
