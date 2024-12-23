package main

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/micro"
)

// PingHandler is the handler for `warehouse.ping`
func PingHandler(_ context.Context, req micro.Request, _ *pgxpool.Pool) {
	_ = req.Respond([]byte("pong"))
}
