package natsutil

import (
	"fmt"

	"github.com/nats-io/nats.go"
)

type Description struct {
	code        string
	description string
}

var (
	InvalidRequest    = Description{"invalid_request", "Failed to deserialize request body"}
	NatsError         = Description{"internal_error", "Failed to publish data to NATS"}
	InsufficientStock = Description{"insufficient_stock", "Not enough stock to fulfill order"}
	CatalogIdNotFound = Description{"not_found", "Failed to find catalog item with given id"}
	MarshalError      = Description{"internal_error", "Failed to serialize response body"}
	SendResponseError = Description{"internal_error", "Failed to send response data"}
	QueryError        = Description{"internal_error", "Failed to query database"}
	KvError           = Description{"internal_error", "Failed to query KV"}
)

func Respond(request *nats.Msg, err Description) {
	_ = request.Respond([]byte(fmt.Sprintf("%s: %s", err.code, err.description)))
	if err == SendResponseError || err == QueryError {
		_ = request.Nak()
	}
}
