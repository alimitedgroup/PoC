package common

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/micro"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// StatefulHandler is a utility type for creating NATS handlers that get passed
// a context.Context, and some shared state that is necessary for handling the request.
//
// It also takes care of setting up observability for each request.
type StatefulHandler struct {
	ctx     context.Context
	handler func(context.Context, micro.Request, *pgxpool.Pool)
	db      *pgxpool.Pool
}

func (h StatefulHandler) Handle(req micro.Request) {
	NumRequests.Add(h.ctx, 1, metric.WithAttributes(
		attribute.String("subject", (req).Subject()),
	))

	// TODO: misurare i tempi di risposta, e salvarli in un histogram

	h.handler(h.ctx, req, h.db)
}

func NewHandler(ctx context.Context, db *pgxpool.Pool, handler func(context.Context, micro.Request, *pgxpool.Pool)) StatefulHandler {
	return StatefulHandler{ctx: ctx, handler: handler, db: db}
}
