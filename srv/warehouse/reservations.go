package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/alimitedgroup/PoC/common"
	"github.com/alimitedgroup/PoC/common/messages"
	"github.com/nats-io/nats.go/jetstream"
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

type reservationState struct {
	sync.Mutex
	s []Reservation
}

func ReservationHandler(ctx context.Context, s *common.Service[warehouseState], req jetstream.Msg) error {
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
		return nil
	}

	meta, err := req.Metadata()
	if err != nil {
		slog.ErrorContext(
			ctx, "Error getting metadata for message",
			"error", err,
			"subject", req.Subject(),
			"message", req.Headers()["Nats-Msg-Id"][0],
		)
		return nil
	}

	reservations := &s.State().reservation

	// reservations MUST be locked
	reservations.s = append(reservations.s, Reservation{
		Reservation: msg,
		seq:         meta.Sequence.Stream,
		ts:          meta.Timestamp,
	})

	return nil
}

func removeReservationsLoop(ctx context.Context, reservations *reservationState) {
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

func PublishReservation(ctx context.Context, reservations *reservationState, js jetstream.JetStream, msg messages.Reservation) error {
	reservations.Lock()
	defer reservations.Unlock()

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal stock update: %w", err)
	}

	_, err = js.Publish(ctx, fmt.Sprintf("reservations.%s", warehouseId), body)
	if err != nil {
		return fmt.Errorf("failed to publish stock update: %w", err)
	}

	return nil
}
