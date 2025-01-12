package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/alimitedgroup/PoC/common"

	"github.com/alimitedgroup/PoC/common/messages"
	"github.com/alimitedgroup/PoC/common/natsutil"
	"github.com/jackc/pgx/v5"
	"github.com/nats-io/nats.go"
)

// PingHandler is the handler for `catalog.ping`
func PingHandler(_ context.Context, _ *common.Service[catalogState], req *nats.Msg) {
	_ = req.Respond([]byte("pong"))
}

// CreateHandler is the handler for `catalog.create`
func CreateHandler(ctx context.Context, s *common.Service[catalogState], req *nats.Msg) {
	var msg messages.CreateCatalogItem
	err := json.Unmarshal(req.Data, &msg)
	if err != nil {
		slog.ErrorContext(ctx, "Error unmarshaling request data", "error", err)
		natsutil.Respond(req, natsutil.InvalidRequest)
		return
	}

	var id int
	err = s.State().db.QueryRow(ctx, "INSERT INTO catalog(name) VALUES ($1) RETURNING id", msg.Name).Scan(&id)
	if err != nil {
		slog.ErrorContext(ctx, "Error inserting catalog item", "error", err)
		natsutil.Respond(req, natsutil.QueryError)
		return
	}

	err = req.Respond([]byte(fmt.Sprintf("%d", id)))
	if err != nil {
		slog.ErrorContext(ctx, "Error sending response to client", "error", err)
	}

	stockUpdateMsg := messages.StockUpdate{{
		GoodId: uint64(id),
		Amount: 0,
	}}

	body, err := json.Marshal(stockUpdateMsg)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to marshal value as JSON", "error", err)
	}

	// TODO: This isn't atomic with write to the database,
	// maybe use Transactional Messaging (listen to the WAL of the database and send the message after the transaction is committed)
	_, err = s.JetStream().PublishMsg(ctx, &nats.Msg{
		Subject: fmt.Sprintf("stock_updates.*"),
		Data:    body,
	})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to publish message", "error", err)
	}
}

// GetHandler is the handler for `catalog.get`
func GetHandler(ctx context.Context, s *common.Service[catalogState], req *nats.Msg) {
	var msg messages.GetCatalogItem
	err := json.Unmarshal(req.Data, &msg)
	if err != nil {
		slog.ErrorContext(ctx, "Error unmarshaling request data", "error", err)
		natsutil.Respond(req, natsutil.InvalidRequest)
		return
	}

	rows, err := s.State().db.Query(ctx, "SELECT id, name FROM catalog WHERE id = $1", msg.Id)
	defer rows.Close()
	if err != nil {
		slog.ErrorContext(ctx, "Error querying catalog table", "error", err)
		natsutil.Respond(req, natsutil.QueryError)
		return
	}

	item, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[messages.CatalogItem])
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		natsutil.Respond(req, natsutil.CatalogIdNotFound)
		return
	} else if err != nil {
		slog.ErrorContext(ctx, "Error collecting catalog item", "error", err)
		natsutil.Respond(req, natsutil.QueryError)
		return
	}

	data, err := json.Marshal(&item)
	if err != nil {
		slog.ErrorContext(ctx, "Error marshaling catalog item", "error", err)
		natsutil.Respond(req, natsutil.MarshalError)
		return
	}

	err = req.Respond(data)
	if err != nil {
		natsutil.Respond(req, natsutil.SendResponseError)
		return
	}
}

// ListHandler is the handler for `catalog.list`
func ListHandler(ctx context.Context, s *common.Service[catalogState], req *nats.Msg) {
	rows, err := s.State().db.Query(ctx, "SELECT id, name FROM catalog")
	defer rows.Close()
	if err != nil {
		slog.ErrorContext(ctx, "Error querying catalog table", "error", err)
		natsutil.Respond(req, natsutil.QueryError)
		return
	}

	res, err := pgx.CollectRows(rows, pgx.RowToStructByName[messages.CatalogItem])
	if err != nil {
		slog.ErrorContext(ctx, "Error collecting catalog items", "error", err)
		natsutil.Respond(req, natsutil.QueryError)
		return
	}

	resBody, err := json.Marshal(res)
	if err != nil {
		slog.ErrorContext(ctx, "Error marshaling catalog items", "error", err)
		natsutil.Respond(req, natsutil.MarshalError)
		return
	}

	err = req.Respond(resBody)
	if err != nil {
		slog.ErrorContext(ctx, "Error marshaling catalog items response", "error", err)
	}
}

// UpdateHandler is the handler for `catalog.update`
func UpdateHandler(ctx context.Context, s *common.Service[catalogState], req *nats.Msg) {
	var msg messages.CatalogItem
	err := json.Unmarshal(req.Data, &msg)
	if err != nil {
		slog.ErrorContext(ctx, "Error unmarshaling request data", "error", err)
		natsutil.Respond(req, natsutil.InvalidRequest)
		return
	}

	_, err = s.State().db.Exec(ctx, "UPDATE catalog SET name = $1 WHERE id = $2", msg.Name, msg.Id)
	if err != nil {
		slog.ErrorContext(ctx, "Error updating catalog item", "error", err)
		natsutil.Respond(req, natsutil.QueryError)
		return
	}

	err = req.Respond([]byte("ok"))
	if err != nil {
		slog.ErrorContext(ctx, "Error sending response to client", "error", err)
	}
}

// DeleteHandler is the handler for `catalog.update`
func DeleteHandler(ctx context.Context, s *common.Service[catalogState], req *nats.Msg) {
	var msg messages.GetCatalogItem
	err := json.Unmarshal(req.Data, &msg)
	if err != nil {
		slog.ErrorContext(ctx, "Error unmarshaling request data", "error", err)
		natsutil.Respond(req, natsutil.InvalidRequest)
		return
	}

	_, err = s.State().db.Exec(ctx, "DELETE FROM catalog WHERE id = $1", msg.Id)
	if err != nil {
		slog.ErrorContext(ctx, "Error updating catalog item", "error", err)
		natsutil.Respond(req, natsutil.QueryError)
		return
	}

	err = req.Respond([]byte("ok"))
	if err != nil {
		slog.ErrorContext(ctx, "Error sending response to client", "error", err)
	}
}
