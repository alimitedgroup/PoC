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
