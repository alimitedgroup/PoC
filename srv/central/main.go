package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	. "magazzino/common"

	"github.com/nats-io/nats.go"
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

func ListenEvents(nc *nats.Conn) *nats.Subscription {
	// Subscribe to a subject
	sub, err := nc.Subscribe("warehouse_events.*", func(m *nats.Msg) {
		var event Event
		err := json.Unmarshal(m.Data, &event)
		if err != nil {
			log.Fatalf("Error unmarshalling event: %v", err)
		}

		switch event.Table {
		case "merce_event":
			var merceEvent AddStockEvent
			err = json.Unmarshal(event.Data.Message, &merceEvent)
			if err != nil {
				log.Fatalf("Error unmarshalling merce event: %v", err)
			}
			log.Printf("Received merce event: %+v\n", merceEvent)

		case "order_event":
			var orderEvent CreateOrderEvent
			err = json.Unmarshal(event.Data.Message, &orderEvent)
			if err != nil {
				log.Fatalf("Error unmarshalling order event: %v", err)
			}
			log.Printf("Received order event: %+v\n", orderEvent)

		}
	})
	if err != nil {
		log.Fatal(err)
	}
	return sub
}

func getRoot(w http.ResponseWriter, r *http.Request) {
	log.Printf("got / request\n")
	io.WriteString(w, "Hello!\n")
}
func getHealth(w http.ResponseWriter, r *http.Request) {
	log.Printf("got /health request\n")
	io.WriteString(w, "OK\n")
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

	sub := ListenEvents(nc)
	defer sub.Unsubscribe()

	http.HandleFunc("/", getRoot)
	http.HandleFunc("/health", getHealth)

	err = http.ListenAndServe(fmt.Sprintf(":%v", listenPort), nil)
	if err != nil {
		log.Fatal(err)
	}

	select {}
}
