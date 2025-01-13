package main

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/alimitedgroup/PoC/common"
	"github.com/alimitedgroup/PoC/common/messages"
	"github.com/nats-io/nats.go/jetstream"
)

func StockUpdateHandler(ctx context.Context, s *common.Service[orderState], req jetstream.Msg) error {
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
		stock.m[row.GoodId] = row.Amount
	}

	slog.InfoContext(
		ctx,
		"Stock updated",
		"goods", len(msg),
	)

	return nil
}
