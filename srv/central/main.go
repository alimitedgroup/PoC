package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/nats-io/nats.go"
)

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
			log.Fatalf("No callback found for event: %+v", event)
		}
		f(event)
	})
	if err != nil {
		log.Fatal(err)
	}
	return sub
}

func main() {
	natsUrl := os.Getenv("NATS_URL")
	listenPort := os.Getenv("LISTEN_PORT")

	// Connect to a NATS server
	nc, err := nats.Connect(natsUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()
	log.Println("Connected to NATS server")

	InitMerceState()

	sub := ListenEvents(nc)
	defer sub.Unsubscribe()

	go startServer(listenPort)

	select {}
}
