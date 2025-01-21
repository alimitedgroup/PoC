package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/alimitedgroup/PoC/common"
	"github.com/alimitedgroup/PoC/common/messages"
	"github.com/nats-io/nats.go/jetstream"
)

func StockUpdateHandler(ctx context.Context, s *common.Service[orderState], req jetstream.Msg) error {
	var msg messages.StockUpdate
	if err := json.Unmarshal(req.Data(), &msg); err != nil {
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

	warehouseId, found := strings.CutPrefix(req.Subject(), "stock_updates.")
	if !found {
		err2 := req.TermWithReason("received message on stock_updates with strange subject")
		if err2 != nil {
			return fmt.Errorf(
				"while handling 'received message on stock_updates with strange subject', another error happened: %w",
				fmt.Errorf("failed to Term message: %w", err2),
			)
		}
		return fmt.Errorf("received message on stock_updates with strange subject: %s", req.Subject())
	}

	// use mutex to protect stock map
	stock.Lock()
	defer stock.Unlock()

	wStock, ok := stock.m[warehouseId]
	if !ok {
		stock.m[warehouseId] = make(map[string]int)
	}

	for _, row := range msg {
		wStock[row.GoodId] = row.Amount
	}

	slog.InfoContext(
		ctx,
		"Stock updated",
		"goods", len(msg),
	)

	return nil
}
