package main

import (
	"encoding/json"
	"log"

	. "magazzino/common"
)

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

func handleMerceEvent(event Event) {
	var merceEvent AddStockEvent
	err := json.Unmarshal(event.Data.Message, &merceEvent)
	if err != nil {
		log.Fatalf("Error unmarshalling merce event: %v", err)
	}
	log.Printf("Received merce event: %+v\n", merceEvent)
}

func handleOrderEvent(event Event) {
	var orderEvent CreateOrderEvent
	err := json.Unmarshal(event.Data.Message, &orderEvent)
	if err != nil {
		log.Fatalf("Error unmarshalling order event: %v", err)
	}
	log.Printf("Received order event: %+v\n", orderEvent)
}

var EventCallbacks = map[string]func(Event){
	"merce_event": handleMerceEvent,
	"order_event": handleOrderEvent,
}
