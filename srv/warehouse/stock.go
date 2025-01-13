package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/alimitedgroup/PoC/common"
	"github.com/alimitedgroup/PoC/common/messages"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// stock contains the currently stocked items inside of the field `s`,
// and the amounts of items that have been reserved, inside the `r` field.
// Each of these fields is a map from good id to stocked (or reserved) amount.
//
// Please note that this singleton should be locked before being used by
// calling its `Lock()` method.
type stockState struct {
	sync.Mutex
	s map[string]int
	r map[string]int
}

func StockUpdateHandler(ctx context.Context, s *common.Service[warehouseState], req jetstream.Msg) error {
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
		return nil
	}

	stock := &s.State().stock

	// use mutex to protect stock map
	stock.Lock()
	defer stock.Unlock()

	for _, row := range msg {
		// stock MUST be locked
		stock.s[row.GoodId] = row.Amount
		stock.r[row.GoodId] = 0
	}

	return nil
}

func SendStockUpdate(ctx context.Context, js jetstream.JetStream, msg *messages.StockUpdate) error {
	body, err := json.Marshal(msg)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to marshal value as JSON", "error", err)
		return err
	}

	_, err = js.PublishMsg(ctx, &nats.Msg{
		Subject: fmt.Sprintf("stock_updates.%s", warehouseId),
		Data:    body,
	})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to publish message", "error", err)
		return err
	}

	return nil
}
