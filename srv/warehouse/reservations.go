package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/alimitedgroup/palestra_poc/common/messages"
	"github.com/alimitedgroup/palestra_poc/common/natsutil"
	"github.com/nats-io/nats.go/jetstream"
	"log/slog"
	"sync"
	"time"
)

type Reservation struct {
	messages.Reservation
	// seq is the sequence number of this reservation in the stream
	seq uint64
	// ts is the timestamp this reservation was published at
	ts time.Time
}

// ReservationTimeout specifies the time after which a reservation will be considered "cancelled"
const ReservationTimeout = 30 * time.Minute

var reservations = struct {
	sync.Mutex
	s []Reservation
}{sync.Mutex{}, make([]Reservation, 0)}

func InitReservations(ctx context.Context, js jetstream.JetStream) error {
	reservations.Lock()
	defer reservations.Unlock()

	_, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     "reservations",
		Subjects: []string{"reservations.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   ReservationTimeout,
	})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create stream", "error", err, "stream", "reservations")
		return fmt.Errorf("failed to create stream: %w", err)
	}

	err = natsutil.ConsumeAll(
		ctx,
		js,
		"reservations",
		jetstream.OrderedConsumerConfig{FilterSubjects: []string{fmt.Sprintf("reservations.%s", warehouseId)}},
		func(msg jetstream.Msg) { ReservationHandler(ctx, msg); _ = msg.Ack() },
	)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to consume stock updates", "error", err, "stream", "stock_updates")
		return fmt.Errorf("failed to consume stock updates: %w", err)
	}

	go removeReservationsLoop(ctx)
	slog.InfoContext(ctx, "Stock updates handled", "stock", stock.s)
	return nil
}

func ReservationHandler(ctx context.Context, req jetstream.Msg) {
	var msg messages.Reservation
	err := json.Unmarshal(req.Data(), &msg)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"Error unmarshalling message",
			"error", err,
			"subject", req.Subject(),
			"message", req.Headers()["Nats-Msg-Id"][0],
		)
		return
	}

	meta, err := req.Metadata()
	if err != nil {
		// TODO
		return
	}

	// reservations MUST be locked
	reservations.s = append(reservations.s, Reservation{
		Reservation: msg,
		seq:         meta.Sequence.Stream,
		ts:          meta.Timestamp,
	})
}

func removeReservationsLoop(ctx context.Context) {
	t := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			reservations.Lock()
			for i, reservation := range reservations.s {
				if reservation.ts.Add(ReservationTimeout).After(time.Now()) {
					reservations.s = append(reservations.s[:i], reservations.s[i+1:]...)
				}
			}
			reservations.Unlock()
		}
	}
}
