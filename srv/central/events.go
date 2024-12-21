package main

import (
	"encoding/json"
	"fmt"
	"log"

	. "magazzino/common"

	"github.com/nats-io/nats.go"
)

func handleMerceEvent(event Event) {
	var merceEvent AddStockEvent
	err := json.Unmarshal(event.Data.Message, &merceEvent)
	if err != nil {
		log.Fatalf("Error unmarshalling merce event: %v", err)
	}
	log.Printf("Received stock update of merce: %v\n", merceEvent.MerceId)

	_, ok := StockOfMerce[merceEvent.MerceId]
	if !ok {
		log.Fatalf("Merce %v not found in stock, inconsistent global state\n", merceEvent.MerceId)
	}
	StockOfMerce[merceEvent.MerceId] += merceEvent.Stock
	log.Printf("increased stock of merce %v to %v\n", merceEvent.MerceId, StockOfMerce[merceEvent.MerceId])
}

func handleOrderEvent(event Event) {
	var orderEvent CreateOrderEvent
	err := json.Unmarshal(event.Data.Message, &orderEvent)
	if err != nil {
		log.Fatalf("Error unmarshalling order event: %v", err)
	}
	log.Printf("Received order: %v\n", orderEvent.OrderId)

	for _, item := range orderEvent.Merci {
		_, ok := StockOfMerce[item.MerceId]
		if !ok {
			log.Fatalf("Merce %v not found in stock, inconsistent global state\n", item.MerceId)
		}
		StockOfMerce[item.MerceId] -= item.Stock
		log.Printf("decreased stock of merce %v to %v\n", item.MerceId, StockOfMerce[item.MerceId])
	}
}

var EventCallbacks = map[string]func(Event){
	"merce_event": handleMerceEvent,
	"order_event": handleOrderEvent,
}

var StockOfMerce = map[int64]int64{}

func InitMerceState() {
	// TODO: read events from catalog service
	StockOfMerce = map[int64]int64{
		1: 0,
		2: 0,
		3: 0,
	}
}

const subjectName = "warehouse_events"

func ListenEvents(nc *nats.Conn) *nats.Subscription {
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
		f(event)
	})
	if err != nil {
		log.Fatal(err)
	}
	return sub
}
