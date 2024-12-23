package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/alimitedgroup/palestra_poc/common/messages"
	"github.com/alimitedgroup/palestra_poc/common/natserr"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/micro"
	"log/slog"
)

// PingHandler is the handler for `catalog.ping`
func PingHandler(_ context.Context, req micro.Request, _ *pgxpool.Pool) {
	_ = req.Respond([]byte("pong"))
}

// GetHandler is the handler for `catalog.get`
func GetHandler(ctx context.Context, req micro.Request, db *pgxpool.Pool) {
	var msg messages.GetCatalogItem
	err := json.Unmarshal(req.Data(), &msg)
	if err != nil {
		slog.ErrorContext(ctx, "Error unmarshaling request data", "error", err)
		natserr.Respond(&req, natserr.InvalidRequest)
		return
	}

	rows, err := db.Query(ctx, "SELECT id, name FROM catalog WHERE id = $1", msg.Id)
	defer rows.Close()
	if err != nil {
		slog.ErrorContext(ctx, "Error querying catalog table", "error", err)
		natserr.Respond(&req, natserr.QueryError)
		return
	}

	item, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[messages.CatalogItem])
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		natserr.Respond(&req, natserr.CatalogIdNotFound)
		return
	} else if err != nil {
		slog.ErrorContext(ctx, "Error collecting catalog item", "error", err)
		natserr.Respond(&req, natserr.QueryError)
		return
	}

	data, err := json.Marshal(&item)
	if err != nil {
		slog.ErrorContext(ctx, "Error marshaling catalog item", "error", err)
		natserr.Respond(&req, natserr.MarshalError)
		return
	}

	err = req.Respond(data)
	if err != nil {
		natserr.Respond(&req, natserr.SendResponseError)
		return
	}
}
