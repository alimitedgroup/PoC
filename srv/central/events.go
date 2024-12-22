package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	. "magazzino/common"

	"github.com/nats-io/nats.go"
)

func handleMerceStockUpdateEvent(ctx context.Context, event Event, db *sql.DB) {
	var merceEvent AddStockEvent
	err := json.Unmarshal(event.Data.Message, &merceEvent)
	if err != nil {
		log.Fatalf("Error unmarshalling merce event: %v", err)
	}
	log.Printf("Received stock update of merce: %v\n", merceEvent.MerceId)

	_, err = db.Exec("UPDATE merce SET stock = stock + $1 WHERE id = $2", merceEvent.Stock, merceEvent.MerceId)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Fatalf("Merce %v not found in stock, inconsistent global state\n", merceEvent.MerceId)
		} else {
			log.Fatalf("Error updating stock in database: %v", err)
		}
	}

	log.Printf("increased stock of merce %v\n", merceEvent.MerceId)
}

func handleCreateOrderEvent(ctx context.Context, event Event, db *sql.DB) {
	var orderEvent CreateOrderEvent
	err := json.Unmarshal(event.Data.Message, &orderEvent)
	if err != nil {
		log.Fatalf("Error unmarshalling order event: %v", err)
	}
	log.Printf("Received order: %v\n", orderEvent.OrderId)

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Error beginning transaction: %v", err)
	}
	defer tx.Rollback()

	for _, item := range orderEvent.Merci {
		_, err := tx.Exec("UPDATE merce SET stock = stock - $1 WHERE id = $2", item.Stock, item.MerceId)
		if err != nil {
			log.Fatalf("Error updating stock in database: %v", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Fatalf("Error committing transaction: %v", err)
	}

	log.Printf("applied order: %v\n", orderEvent.OrderId)
}

var EventCallbacks = map[string]func(context.Context, Event, *sql.DB){
	"merce_stock_update_event": handleMerceStockUpdateEvent,
	"create_order_event":       handleCreateOrderEvent,
}

const subjectName = "warehouse_events"

func ListenEvents(nc *nats.Conn, db *sql.DB) *nats.Subscription {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "db", db)

	// Subscribe to a subject
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
