package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	. "magazzino/common"

	_ "github.com/jackc/pgx/stdlib"
	"github.com/nats-io/nats.go"
)

func fakeOperations(db *sql.DB) {
	merci := []MerceStock{
		{
			MerceId: 1,
			Stock:   1,
		},
	}
	orderNote := "test order note"
	var newStock int64 = 10

	for {
		for _, merce := range merci {
			err := InsertStockMerce(db, merce.MerceId, newStock)
			if err != nil {
				log.Fatal(err)
			}
		}

		err := InsertOrder(db, orderNote, merci)
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(1 * time.Second)
	}
}

func main() {
	dbConnStr := os.Getenv("DB_URL")
	listenPort := os.Getenv("LISTEN_PORT")
	natsUrl := os.Getenv("NATS_URL")

	// Connect to the database
	db, err := sql.Open("pgx", dbConnStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Test the connection
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	log.Println("Connected to PostgreSQL!")

	// Connect to a NATS server
	nc, err := nats.Connect(natsUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()
	log.Println("Connected to NATS server")

	go startServer(listenPort)
	sub := ListenEvents(nc, db)
	defer sub.Unsubscribe()
	// go fakeOperations(db)

	select {}
}
