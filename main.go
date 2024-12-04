package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/nats-io/nats.go"
)

const dbConnStr = "host=postgres user=postgres password=postgres dbname=database sslmode=disable"

func listenEvents(nc *nats.Conn) *nats.Subscription {
	// Subscribe to a subject
	sub, err := nc.Subscribe("warehouse_events.*", func(m *nats.Msg) {
		log.Printf("Received a message: %s\n", string(m.Data))
	})
	if err != nil {
		log.Fatal(err)
	}
	return sub
}

func insertStockMerce(db *sql.DB) {
	type AddStockEvent struct {
		MerceId int64 `json:"merce_id"`
		Stock   int64 `json:"stock"`
	}
	addStockEvent := AddStockEvent{
		MerceId: 1,
		Stock:   10,
	}
	var stmt *sql.Stmt
	var res sql.Result
	var rowCount int64
	var err error

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	stmt, err = tx.Prepare("UPDATE merce SET stock=stock+$2 WHERE id = $1")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	res, err = stmt.Exec(addStockEvent.MerceId, addStockEvent.Stock)
	if err != nil {
		log.Fatal(err)
	}
	rowCount, err = res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Updated %d row(s)\n", rowCount)

	stmt, err = tx.Prepare("INSERT INTO merce_event (message) VALUES ($1)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	data, err := json.Marshal(addStockEvent)
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
	log.Printf("Updated %d row(s)\n", rowCount)

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

func insertOrder(db *sql.DB) {
	type OrderMerce struct {
		MerceId int64 `json:"merce_id"`
		Stock   int   `json:"stock"`
	}
	type OrderEvent struct {
		OrderId int64        `json:"order_id"`
		Note    string       `json:"note"`
		Merci   []OrderMerce `json:"merci"`
	}

	orderEvent := OrderEvent{
		Note: "test order",
		Merci: []OrderMerce{
			{
				MerceId: 1,
				Stock:   3,
			},
		},
	}

	var stmt *sql.Stmt
	var res sql.Result
	var rowCount int64
	var err error

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	stmt, err = tx.Prepare("INSERT INTO orders (note) VALUES ($1) RETURNING id")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	err = stmt.QueryRow(orderEvent.Note).Scan(&orderEvent.OrderId)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Inserted order with id: %d\n", orderEvent.OrderId)

	stmt, err = tx.Prepare("INSERT INTO order_merce (order_id, merce_id, stock) VALUES ($1, $2, $3)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	res, err = stmt.Exec(orderEvent.OrderId, orderEvent.Merci[0].MerceId, orderEvent.Merci[0].Stock)
	if err != nil {
		log.Fatal(err)
	}

	rowCount, err = res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Inserted %d row(s)\n", rowCount)

	stmt, err = tx.Prepare("UPDATE merce SET stock=stock-$2 WHERE id = $1")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	res, err = stmt.Exec(orderEvent.Merci[0].MerceId, orderEvent.Merci[0].Stock)
	if err != nil {
		log.Fatal(err)
	}
	rowCount, err = res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Updated %d row(s)\n", rowCount)

	stmt, err = tx.Prepare("INSERT INTO order_event (message) VALUES ($1)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

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
	// Connect to a NATS server
	nc, err := nats.Connect("nats://nats:4222")
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()
	log.Println("Connected to NATS server")
	sub := listenEvents(nc)
	defer sub.Unsubscribe()

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
	fmt.Println("Connected to PostgreSQL!")

	for {
		insertStockMerce(db)
		insertOrder(db)
		time.Sleep(1 * time.Second)
	}
}
