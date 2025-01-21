package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

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

	// var tmpCreate = req

	var state = s.State()

	state.stock.Lock()
	defer state.stock.Unlock()

	var order messages.OrderCreated
	order.ID = uuid.New()
	order.Items = make([]messages.OrderCreatedItem, 0)

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
				if usedStock[goodId] == nil {
					usedStock[goodId] = make(map[string]int)
				}
				usedStock[goodId][warehouseId] = used
			}
		}
	}

	// if there is still remaining stock, then we don't have enough stock to fulfill the order
	if totalRemainingStock > 0 {
		natsutil.Respond(msg, natsutil.InsufficientStock)
		return
	}

	// create the order message
	for _, item := range req.Items {
		var parts = make([]messages.OrderCreatedItemPart, 0)

		for warehouseId, amount := range usedStock[item.GoodId] {
			parts = append(parts, messages.OrderCreatedItemPart{
				WarehouseId: warehouseId,
				Amount:      amount,
			})
		}

		order.Items = append(order.Items, messages.OrderCreatedItem{
			GoodId: item.GoodId,
			Parts:  parts,
		})
	}

	payload, err := json.Marshal(order)
	if err != nil {
		slog.ErrorContext(ctx, "Error marshaling response data", "error", err)
		natsutil.Respond(msg, natsutil.MarshalError)
		return
	}

	// NOTE: don't update the stock here, it should be done in the warehouse service that will send back a stock_update event

	if _, err = s.JetStream().PublishMsg(ctx, &nats.Msg{
		Subject: "orders",
		Data:    payload,
	}); err != nil {
		slog.ErrorContext(ctx, "Error publishing response", "error", err)
		natsutil.Respond(msg, natsutil.NatsError)
		return
	}

	_ = msg.Respond([]byte(fmt.Sprintf("order created: %s", order.ID)))
}
