package main

import (
	"context"
	"encoding/json"

	"github.com/alimitedgroup/palestra_poc/common/messages"
	"github.com/nats-io/nats.go"
)

// PingHandler is the handler for `warehouse.ping`
func PingHandler(msg *nats.Msg) {
	_ = msg.Respond([]byte("pong"))
}

// ReserveHandler is the handler for `warehouse.reserve`
func ReserveHandler(ctx context.Context, req *nats.Msg) {
	var msg messages.ReserveStock
	err := json.Unmarshal(req.Data, &msg)
	if err != nil {
		_ = req.Respond([]byte(err.Error()))
		return // TODO
	}

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
			js,
			messages.Reservation{
				ID:            msg.ID,
				ReservedStock: msg.RequestedStock,
			},
		)

		for _, s := range msg.RequestedStock {
			stock.r[s.GoodId] += s.Amount
		}

		_ = req.Respond([]byte("ok"))
	} else {
		_ = req.Respond([]byte("not enough stock"))
	}
}
