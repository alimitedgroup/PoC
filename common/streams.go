package common

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
)

var ReservationStreamConfig = jetstream.StreamConfig{
	Name:     "reservations",
	Subjects: []string{"reservations.>"},
	Storage:  jetstream.FileStorage,
}

var StockUpdatesStreamConfig = jetstream.StreamConfig{
	Name:     "stock_updates",
	Subjects: []string{"stock_updates.>"},
	Storage:  jetstream.FileStorage,
}

var OdersStreamConfig = jetstream.StreamConfig{
	Name:     "orders",
	Subjects: []string{"orders.>"},
	Storage:  jetstream.FileStorage,
}

func CreateStream(ctx context.Context, js jetstream.JetStream, cfg jetstream.StreamConfig) error {
	_, err := js.CreateStream(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}
	return nil
}
