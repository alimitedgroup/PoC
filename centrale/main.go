package main

import (
	"log"

	"github.com/nats-io/nats.go"
)

func ListenEvents(nc *nats.Conn) *nats.Subscription {
	// Subscribe to a subject
	sub, err := nc.Subscribe("warehouse_events.*", func(m *nats.Msg) {
		log.Printf("Received a message: %s\n", string(m.Data))
	})
	if err != nil {
		log.Fatal(err)
	}
	return sub
}

func main() {
	// Connect to a NATS server
	nc, err := nats.Connect("nats://nats:4222")
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()
	log.Println("Connected to NATS server")
	sub := ListenEvents(nc)
	defer sub.Unsubscribe()

	select {}
}
