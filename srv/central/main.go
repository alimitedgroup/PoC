package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/jackc/pgx/stdlib"
	"github.com/nats-io/nats.go"
)

func main() {
	natsUrl := os.Getenv("NATS_URL")
	listenPort := os.Getenv("LISTEN_PORT")
	dbConnStr := os.Getenv("DB_URL")

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

	InitCatalog(db)

	sub := ListenEvents(nc)
	defer sub.Unsubscribe()

	go startServer(listenPort)

	select {}
}
