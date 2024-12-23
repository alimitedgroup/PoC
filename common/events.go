package common

import (
	"encoding/json"
)

type AddStockEvent struct {
	MerceId int64 `json:"merce_id"`
	Stock   int64 `json:"stock"`
}

type MerceStock struct {
	MerceId int64 `json:"merce_id"`
	Stock   int64 `json:"stock"`
}

type CreateOrderEvent struct {
	OrderId int64        `json:"order_id"`
	Note    string       `json:"note"`
	Merci   []MerceStock `json:"merci"`
}

type CreateMerceEvent struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Event struct {
	Id         string `json:"id"`
	Schema     string `json:"schema"`
	Table      string `json:"table"`
	Action     string `json:"action"`
	CommitTime string `json:"commitTime"`
	Data       struct {
		CreatedAt string          `json:"created_at"`
		Id        int64           `json:"id"`
		Message   json.RawMessage `json:"message"`
	} `json:"data"`
	DataOld json.RawMessage `json:"dataOld"` // only for Action = "UPDATE"
}
