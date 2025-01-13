package main

import (
	"context"

	"github.com/alimitedgroup/PoC/common"

	"github.com/nats-io/nats.go"
)

// PingHandler is the handler for `order.ping`
func PingHandler(_ context.Context, _ *common.Service[orderState], msg *nats.Msg) {
	_ = msg.Respond([]byte("pong"))
}
