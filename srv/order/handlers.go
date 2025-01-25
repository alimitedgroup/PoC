package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/alimitedgroup/PoC/common"
	"github.com/alimitedgroup/PoC/common/messages"
	"github.com/alimitedgroup/PoC/common/natsutil"
	"github.com/google/uuid"

	"github.com/nats-io/nats.go"
)

// PingHandler is the handler for `order.ping`
func PingHandler(_ context.Context, _ *common.Service[orderState], msg *nats.Msg) {
	_ = msg.Respond([]byte("pong"))
}

// CreateOrderHandler is the handler for `order.create`
func CreateOrderHandler(ctx context.Context, s *common.Service[orderState], msg *nats.Msg) {
	var req messages.CreateOrder
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		slog.ErrorContext(ctx, "Error unmarshaling request data", "error", err)
		natsutil.Respond(msg, natsutil.InvalidRequest)
		return
	}

	var state = s.State()

	state.stock.Lock()
	defer state.stock.Unlock()

	var remainingStock = make(map[string]int)
	var totalRemainingStock = 0
	for _, v := range req.Items {
		remainingStock[v.GoodId] = v.Amount
		totalRemainingStock += v.Amount
	}
	var usedStock = make(map[string]map[string]int)

	// NOTE: naive implementation, use all the stock of the warehouse to fullfill the order, it not enought
	// check each warehouse for stock
	for warehouseId, m := range state.stock.m {
		if totalRemainingStock == 0 {
			break
		}

		for goodId, amount := range m {
			used := 0
			if remainingStock[goodId] > 0 {
				if amount >= remainingStock[goodId] {
					used = remainingStock[goodId]
				} else {
					used = amount
				}
			}
			// will use "used" amount of this warehouse for the order
			remainingStock[goodId] -= used
			totalRemainingStock -= used
			if used > 0 {
				if usedStock[warehouseId] == nil {
					usedStock[warehouseId] = make(map[string]int)
				}
				usedStock[warehouseId][goodId] = used
			}
		}
	}

	// if there is still remaining stock, then we don't have enough stock to fulfill the order
	if totalRemainingStock > 0 {
		natsutil.Respond(msg, natsutil.InsufficientStock)
		return
	}

	// create and send all the reservations messages
	// TODO: use concurrency
	var warehouseReservationIds = make(map[string]uuid.UUID)
	for warehouseId, m := range usedStock {
		var items = make([]messages.ReserveStockItem, 0)
		for goodId, amount := range m {
			items = append(items, messages.ReserveStockItem{
				GoodId: goodId,
				Amount: amount,
			})
		}

		id := uuid.New()
		warehouseReservationIds[warehouseId] = id
		payload, err := json.Marshal(messages.ReserveStock{
			ID:             id,
			RequestedStock: items,
		})
		if err != nil {
			slog.ErrorContext(ctx, "Error marshaling response", "error", err)
			natsutil.Respond(msg, natsutil.NatsError)
			return
		}

		r, err := s.NatsConn().RequestMsg(&nats.Msg{
			Subject: fmt.Sprintf("warehouse.reserve.%s", warehouseId),
			Data:    payload,
		}, time.Second*3)
		if err != nil {
			slog.ErrorContext(ctx, "Error sending the reserve message", "error", err)
			natsutil.Respond(msg, natsutil.NatsError)
			return
		}

		if err = r.Ack(); err != nil {
			slog.ErrorContext(ctx, "Error acking the response", "error", err)
			natsutil.Respond(msg, natsutil.NatsError)
			return
		}
	}

	var order = messages.OrderCreated{
		ID:         uuid.New(),
		Warehouses: make([]messages.OrderCreateWarehouse, 0),
	}

	for warehouseId, m := range usedStock {
		var warehouseItems = make([]messages.OrderCreatedItem, 0)
		for goodId, amount := range m {
			warehouseItems = append(warehouseItems, messages.OrderCreatedItem{
				GoodId: goodId,
				Amount: amount,
			})
		}
		reservationId := warehouseReservationIds[warehouseId]
		order.Warehouses = append(order.Warehouses, messages.OrderCreateWarehouse{
			WarehouseId:   warehouseId,
			ReservationId: reservationId,
			Parts:         warehouseItems,
		})
	}

	payload, err := json.Marshal(order)
	if err != nil {
		slog.ErrorContext(ctx, "Error marshaling response", "error", err)
		natsutil.Respond(msg, natsutil.NatsError)
		return
	}

	_, err = s.JetStream().Publish(ctx, "orders", payload)
	if err != nil {
		slog.ErrorContext(ctx, "Error sending the order created message", "error", err)
		natsutil.Respond(msg, natsutil.NatsError)
		return
	}

	// NOTE: don't update the stock here, it should be done in the warehouse service that will send back a stock_update event

	_ = msg.Respond([]byte(fmt.Sprintf("reservations sent")))
}
