package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
)

const dbConnStr = "host=postgres user=postgres password=postgres dbname=database sslmode=disable"

func listenEvents() {
	// Connect to a NATS server
	nc, err := nats.Connect("nats://nats:4222")
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	log.Println("Connected to NATS server")
	// Subscribe to a subject
	sub, err := nc.Subscribe("warehouse_events.*", func(m *nats.Msg) {
		log.Printf("Received a message: %s\n", string(m.Data))
	})
	if err != nil {
		log.Fatal(err)
	}
	defer sub.Unsubscribe()
}

func insertOrder() {
	type OrderMerce struct {
		MerceId int64 `json:"merce_id"`
		Stock   int   `json:"stock"`
	}
	type OrderEvent struct {
		OrderId int64        `json:"order_id"`
		Note    string       `json:"note"`
		Merci   []OrderMerce `json:"merci"`
	}

	var stmt *sql.Stmt
	var res sql.Result
	var rowCount int64
	var err error

	// Connect to the database
	db, err := sql.Open("postgres", dbConnStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Test the connection
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected to PostgreSQL!")

	// Example Query: Insert using a prepared statement
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	stmt, err = tx.Prepare("INSERT INTO orders (note) VALUES ($1) RETURNING id")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	var orderId int64

	err = stmt.QueryRow("test order").Scan(&orderId)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Inserted order with id: %d\n", orderId)

	stmt, err = tx.Prepare("INSERT INTO order_merce (order_id, merce_id, stock) VALUES ($1, $2, $3)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	res, err = stmt.Exec(orderId, 1, 10)
	if err != nil {
		log.Fatal(err)
	}

	rowCount, err = res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Inserted %d row(s)\n", rowCount)

	stmt, err = tx.Prepare("INSERT INTO order_event (message) VALUES ($1)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	orderEvent := OrderEvent{
		OrderId: 1,
		Note:    "test order",
		Merci: []OrderMerce{
			{
				MerceId: 1,
				Stock:   10,
			},
		},
	}
	data, err := json.Marshal(orderEvent)
	if err != nil {
		log.Fatal(err)
	}

	res, err = stmt.Exec(data)
	if err != nil {
		log.Fatal(err)
	}

	rowCount, err = res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Inserted %d row(s)\n", rowCount)

	defer func() {
		if err != nil {
			log.Println("Rolling back transaction due to error...")
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Fatalf("Failed to rollback transaction: %v", rollbackErr)
			}
			return
		}
		// Commit the transaction if no error
		if commitErr := tx.Commit(); commitErr != nil {
			log.Fatalf("Failed to commit transaction: %v", commitErr)
		}
		log.Println("Transaction committed successfully.")
	}()
}

func main() {
	go listenEvents()

	for {
		insertOrder()
		time.Sleep(10 * time.Second)
	}
}
