package natsutil

import (
	"context"
	"fmt"
	"github.com/nats-io/nats.go/jetstream"
	"log/slog"
	"time"
)

// ConsumeAll is a helper to consume all messages from a stream, and then stop consuming them
func ConsumeAll(
	ctx context.Context,
	js jetstream.JetStream,
	stream string,
	cfg jetstream.OrderedConsumerConfig,
	callback func(msg jetstream.Msg),
) error {
	// `OrderedConsumer`s are ephemeral, and so don't need to be deleted
	consumer, err := js.OrderedConsumer(ctx, stream, cfg)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create OrderedConsumer", "error", err, "stream", stream)
		return fmt.Errorf("failed to create OrderedConsumer: %w", err)
	}

	// Start consuming all messages
	sub, err := consumer.Consume(callback)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to consume messages", "error", err, "stream", stream)
		return fmt.Errorf("failed to consume messages: %w", err)
	}

	// Loop until we consumed all messages, or the context is done
	t := time.NewTicker(250 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			sub.Stop()
			return ctx.Err()
		case <-t.C:
			info, err := consumer.Info(ctx)
			if err != nil {
				slog.ErrorContext(ctx, "Failed to get consumer info", "error", err, "stream", stream)
				return fmt.Errorf("failed to get consumer info: %w", err)
			}

			if (*info).NumPending == 0 {
				sub.Drain()
				return nil
			}
		}
	}
}
