package main

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/alimitedgroup/PoC/common"
	"github.com/google/uuid"

	"github.com/alimitedgroup/PoC/common/messages"
	"github.com/alimitedgroup/PoC/common/natsutil"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// PingHandler is the handler for `catalog.ping`
func PingHandler(_ context.Context, s *common.Service[catalogState], req *nats.Msg) {
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

	id := uuid.New().String()
	body, err := json.Marshal(messages.CatalogItem{
		Id:   id,
		Name: msg.Name,
	})
	if err != nil {
		slog.ErrorContext(ctx, "Error marshaling catalog item", "error", err)
		natsutil.Respond(req, natsutil.MarshalError)
		return
	}

	_, err = s.State().kv.Put(ctx, id, body)
	if err != nil {
		slog.ErrorContext(ctx, "Error storing catalog item in KV", "error", err)
		natsutil.Respond(req, natsutil.KvError)
		return
	}

	err = req.Respond([]byte(id))
	if err != nil {
		slog.ErrorContext(ctx, "Error sending response to client", "error", err)
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

	v, err := s.State().kv.Get(ctx, msg.Id)
	if err != nil {
		slog.ErrorContext(ctx, "Error getting catalog item", "error", err)
		natsutil.Respond(req, natsutil.KvError)
		return
	}

	var item messages.CatalogItem
	err = json.Unmarshal(v.Value(), &item)
	if err != nil {
		slog.ErrorContext(ctx, "Error unmarshaling catalog item", "error", err)
		natsutil.Respond(req, natsutil.MarshalError)
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
	w, err := s.State().kv.WatchAll(ctx, jetstream.IgnoreDeletes())
	if err != nil {
		slog.ErrorContext(ctx, "Error watching keys", "error", err)
		natsutil.Respond(req, natsutil.KvError)
		return
	}

	res := make([]messages.CatalogItem, 0)
	updates := w.Updates()

	for v := range updates {
		var item messages.CatalogItem
		val := v.Value()
		slog.Info("Got catalog item", "value", string(val))
		err = json.Unmarshal(val, &item)
		if err != nil {
			slog.ErrorContext(ctx, "Error unmarshaling catalog item", "error", err)
			natsutil.Respond(req, natsutil.MarshalError)
			return
		}
		slog.Info("Got catalog item", "item", item)
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

	body, err := json.Marshal(msg)
	if err != nil {
		slog.ErrorContext(ctx, "Error marshaling catalog item", "error", err)
		natsutil.Respond(req, natsutil.MarshalError)
		return
	}

	_, err = s.State().kv.Put(ctx, msg.Id, body)
	if err != nil {
		slog.ErrorContext(ctx, "Error storing catalog item in KV", "error", err)
		natsutil.Respond(req, natsutil.KvError)
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

	err = s.State().kv.Delete(ctx, msg.Id)
	if err != nil {
		slog.ErrorContext(ctx, "Error deleting catalog item", "error", err)
		natsutil.Respond(req, natsutil.KvError)
		return
	}

	err = req.Respond([]byte("ok"))
	if err != nil {
		slog.ErrorContext(ctx, "Error sending response to client", "error", err)
	}
}
