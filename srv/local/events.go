package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"

	. "magazzino/common"
)

const subjectName = "warehouse_events"

var EventCallbacks = map[string]func(context.Context, Event, *sql.DB){
	"create_merce_event": handleCreateMerceEvent,
}

func handleCreateMerceEvent(ctx context.Context, event Event, db *sql.DB) {
	repo := ctx.Value("repo").(*Repo)
	if repo == nil {
		log.Fatalf("Error getting repo from context")
	}

	var createMerceEvent CreateMerceEvent
	err := json.Unmarshal(event.Data.Message, &createMerceEvent)
	if err != nil {
		log.Fatalf("Error unmarshalling create merce event: %v", err)
	}
	log.Printf("Received create merce event: %v\n", createMerceEvent.Id)

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Error beginning transaction: %v", err)
	}

	err = repo.CreateMerce(tx, createMerceEvent.Id, createMerceEvent.Name, createMerceEvent.Description)
	if err != nil {
		tx.Rollback()
		log.Fatalf("Error creating merce: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		log.Fatalf("Error committing transaction: %v", err)
	}

}

func ListenEvents(nc *nats.Conn, db *sql.DB) *nats.Subscription {
	// Subscribe to a subject
	ctx := context.Background()
	ctx = context.WithValue(ctx, "repo", &Repo{})

	sub, err := nc.Subscribe(fmt.Sprintf("%v.*", subjectName), func(m *nats.Msg) {
		var event Event
		err := json.Unmarshal(m.Data, &event)
		if err != nil {
			log.Fatalf("Error unmarshalling event: %v", err)
		}

		f, ok := EventCallbacks[event.Table]
		if !ok {
			log.Printf("No callback found for event: %v\n", event.Table)
			return
		}
		f(ctx, event, db)
	})
	if err != nil {
		log.Fatal(err)
	}
	return sub
}
