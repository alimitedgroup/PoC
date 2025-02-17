package main

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/alimitedgroup/PoC/common"
	"github.com/alimitedgroup/PoC/common/messages"
	"github.com/nats-io/nats.go"
)

// PingHandler is the handler for `warehouse.ping`
func PingHandler(_ context.Context, _ *common.Service[warehouseState], req *nats.Msg) {
	_ = req.Respond([]byte("pong"))
}

func convertToReservationItems(items []messages.ReserveStockItem) []messages.ReservationItem {
	reservationItems := make([]messages.ReservationItem, len(items))
	for i, item := range items {
		reservationItems[i] = messages.ReservationItem{
			GoodId: item.GoodId,
			Amount: item.Amount,
		}
	}
	return reservationItems
}

// ReserveHandler is the handler for `warehouse.reserve`
func ReserveHandler(ctx context.Context, s *common.Service[warehouseState], req *nats.Msg) {
	var msg messages.ReserveStock
	err := json.Unmarshal(req.Data, &msg)
	if err != nil {
		_ = req.Respond([]byte(err.Error()))
		return
	}

	stock := &s.State().stock

	stock.Lock()
	defer stock.Unlock()

	// Check whether the reservation request can be satisfied
	ok := true
	for _, s := range msg.RequestedStock {
		// TODO: check if ok, and do something otherwise (crash?)
		current, _ := stock.s[s.GoodId]
		reserved, _ := stock.r[s.GoodId]
		if current-reserved < s.Amount {
			ok = false
			break
		}
	}

	if ok {
		// If the reservation request can be satisfied...
		err = PublishReservation(
			ctx,
			&s.State().reservation,
			s.JetStream(),
			messages.Reservation{
				ID:            msg.ID,
				ReservedStock: convertToReservationItems(msg.RequestedStock),
			},
		)
		if err != nil {
			slog.ErrorContext(
				ctx,
				"Error publishing reservation",
				"error", err,
				"subject", req.Subject,
				"message", req.Header["Nats-Msg-Id"][0],
			)
			_ = req.Respond([]byte("error"))
			return
		}

		for _, s := range msg.RequestedStock {
			stock.r[s.GoodId] += s.Amount
		}

		_ = req.Respond([]byte("ok"))
	} else {
		_ = req.Respond([]byte("not enough stock"))
	}

}

func AddStockHandler(ctx context.Context, s *common.Service[warehouseState], req *nats.Msg) {
	var msg messages.StockUpdate
	err := json.Unmarshal(req.Data, &msg)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"Error unmarshalling message",
			"error", err,
			"subject", req.Subject,
			"message", req.Header["Nats-Msg-Id"][0],
		)
	}

	slog.DebugContext(ctx, "Received stock add request", "msg", msg)

	stock := &s.State().stock

	stock.Lock()
	defer stock.Unlock()

	// msg contains only increment in stock quantity, transform to absolute values using the stock state
	for i, row := range msg {
		msg[i].Amount += stock.s[row.GoodId]
	}

	err = SendStockUpdate(ctx, s.JetStream(), &msg)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"Error sending stock update",
			"error", err,
			"subject", req.Subject,
			"message", req.Header["Nats-Msg-Id"][0],
		)
		return
	}

	for _, row := range msg {
		// stock MUST be locked
		stock.s[row.GoodId] = row.Amount
	}

	_ = req.Respond([]byte("ok"))
}
