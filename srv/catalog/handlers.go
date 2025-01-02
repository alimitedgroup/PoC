package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// CreateHandler is the handler for `catalog.create`
func CreateHandler(ctx context.Context, req micro.Request, db *pgxpool.Pool) {
	var msg messages.CreateCatalogItem
	err := json.Unmarshal(req.Data(), &msg)
	if err != nil {
		slog.ErrorContext(ctx, "Error unmarshaling request data", "error", err)
		natserr.Respond(&req, natserr.InvalidRequest)
		return
	}

	var id int
	err = db.QueryRow(ctx, "INSERT INTO catalog(name) VALUES ($1) RETURNING id", msg.Name).Scan(&id)
	if err != nil {
		slog.ErrorContext(ctx, "Error inserting catalog item", "error", err)
		natserr.Respond(&req, natserr.QueryError)
		return
	}

	err = req.Respond([]byte(fmt.Sprintf("%d", id)))
	if err != nil {
		slog.ErrorContext(ctx, "Error sending response to client", "error", err)
	}
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

// ListHandler is the handler for `catalog.list`
func ListHandler(ctx context.Context, req micro.Request, db *pgxpool.Pool) {
	rows, err := db.Query(ctx, "SELECT id, name FROM catalog")
	defer rows.Close()
	if err != nil {
		slog.ErrorContext(ctx, "Error querying catalog table", "error", err)
		natserr.Respond(&req, natserr.QueryError)
		return
	}

	res, err := pgx.CollectRows(rows, pgx.RowToStructByName[messages.CatalogItem])
	if err != nil {
		slog.ErrorContext(ctx, "Error collecting catalog items", "error", err)
		natserr.Respond(&req, natserr.QueryError)
		return
	}

	err = req.RespondJSON(res)
	if err != nil {
		slog.ErrorContext(ctx, "Error marshaling catalog items response", "error", err)
	}
}

// UpdateHandler is the handler for `catalog.update`
func UpdateHandler(ctx context.Context, req micro.Request, db *pgxpool.Pool) {
	var msg messages.CatalogItem
	err := json.Unmarshal(req.Data(), &msg)
	if err != nil {
		slog.ErrorContext(ctx, "Error unmarshaling request data", "error", err)
		natserr.Respond(&req, natserr.InvalidRequest)
		return
	}

	_, err = db.Exec(ctx, "UPDATE catalog SET name = $1 WHERE id = $2", msg.Name, msg.Id)
	if err != nil {
		slog.ErrorContext(ctx, "Error updating catalog item", "error", err)
		natserr.Respond(&req, natserr.QueryError)
		return
	}

	err = req.Respond([]byte("ok"))
	if err != nil {
		slog.ErrorContext(ctx, "Error sending response to client", "error", err)
	}
}

// DeleteHandler is the handler for `catalog.update`
func DeleteHandler(ctx context.Context, req micro.Request, db *pgxpool.Pool) {
	var msg messages.GetCatalogItem
	err := json.Unmarshal(req.Data(), &msg)
	if err != nil {
		slog.ErrorContext(ctx, "Error unmarshaling request data", "error", err)
		natserr.Respond(&req, natserr.InvalidRequest)
		return
	}

	_, err = db.Exec(ctx, "DELETE FROM catalog WHERE id = $1", msg.Id)
	if err != nil {
		slog.ErrorContext(ctx, "Error updating catalog item", "error", err)
		natserr.Respond(&req, natserr.QueryError)
		return
	}

	err = req.Respond([]byte("ok"))
	if err != nil {
		slog.ErrorContext(ctx, "Error sending response to client", "error", err)
	}
}
