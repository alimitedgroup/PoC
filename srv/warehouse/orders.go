package main

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/alimitedgroup/PoC/common"
	"github.com/alimitedgroup/PoC/common/messages"
	"github.com/nats-io/nats.go/jetstream"
)

func OrdersCreateHandler(ctx context.Context, s *common.Service[warehouseState], req jetstream.Msg) error {
	var msg messages.OrderCreated
	if err := json.Unmarshal(req.Data(), &msg); err != nil {
		slog.ErrorContext(
			ctx,
			"Error unmarshalling message",
			"error", err,
			"subject", req.Subject(),
			"message", req.Headers()["Nats-Msg-Id"][0],
		)
		return err
	}

	var currentWarehouseRequest *messages.OrderCreateWarehouse = nil
	for _, item := range msg.Warehouses {
		if item.WarehouseId == warehouseId {
			currentWarehouseRequest = &item
			break
		}
	}
	if currentWarehouseRequest == nil {
		// This warehouse is not responsible for this order
		return nil
	}

	reserv := &s.State().reservation
	stock := &s.State().stock

	reserv.Lock()
	defer reserv.Unlock()
	stock.Lock()
	defer stock.Unlock()

	// search for the reservation id
	// TODO: make "s" a map
	var reservation *Reservation = nil
	for i, r := range reserv.s {
		if r.Reservation.ID == currentWarehouseRequest.ReservationId {
			reservation = &reserv.s[i]
			break
		}
	}
	// TODO: handle this (?)
	if reservation == nil {
		slog.ErrorContext(ctx, "Reservation expired", "reservation_id", currentWarehouseRequest.ReservationId)
		return nil
	}

	stockUpdate := messages.StockUpdate(make([]messages.StockUpdateItem, 0, len(currentWarehouseRequest.Parts)))
	for _, item := range reservation.ReservedStock {
		// update stock and reservation state
		stock.r[item.GoodId] -= item.Amount
		stock.s[item.GoodId] -= item.Amount
		// add the updated stock state to the stockUpdate message
		stockUpdate = append(stockUpdate, messages.StockUpdateItem{
			GoodId: item.GoodId,
			Amount: stock.s[item.GoodId],
		})
	}

	// send the stock update message to the stream
	if err := SendStockUpdate(ctx, s.JetStream(), &stockUpdate); err != nil {
		slog.ErrorContext(
			ctx,
			"Error sending stock update",
			"error", err,
			"subject", req.Subject(),
			"message", req.Headers()["Nats-Msg-Id"][0],
		)
		return nil
	}

	return nil
}
