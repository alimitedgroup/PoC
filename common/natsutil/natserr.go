package natsutil

import "github.com/nats-io/nats.go/micro"

type Description struct {
	code        string
	description string
}

var (
	InvalidRequest    = Description{"invalid_request", "Failed to deserialize request body"}
	CatalogIdNotFound = Description{"not_found", "Failed to find catalog item with given id"}
	MarshalError      = Description{"internal_error", "Failed to serialize response body"}
	SendResponseError = Description{"internal_error", "Failed to send response data"}
	QueryError        = Description{"internal_error", "Failed to query database"}
)

func Respond(request *micro.Request, err Description) {
	_ = (*request).Error(err.code, err.description, []byte{})
}
