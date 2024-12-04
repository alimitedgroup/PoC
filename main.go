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

type AddStockEvent struct {
	MerceId int64 `json:"merce_id"`
	Stock   int64 `json:"stock"`
}

type OrderMerce struct {
	MerceId int64 `json:"merce_id"`
	Stock   int64 `json:"stock"`
}
type CreateOrderEvent struct {
	OrderId int64        `json:"order_id"`
	Note    string       `json:"note"`
	Merci   []OrderMerce `json:"merci"`
}

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

type Repo struct{}

func (r *Repo) CreateOrder(tx *sql.Tx, note string) int64 {
	var orderId int64
	stmt, err := tx.Prepare("INSERT INTO orders (note) VALUES ($1) RETURNING id")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	err = stmt.QueryRow(note).Scan(&orderId)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Inserted order with id: %d\n", orderId)
	return orderId
}

func (r *Repo) InsertOrderMerce(tx *sql.Tx, orderId int64, merceId int64, stock int64) {
	stmt, err := tx.Prepare("INSERT INTO order_merce (order_id, merce_id, stock) VALUES ($1, $2, $3)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(orderId, merceId, stock)
	if err != nil {
		log.Fatal(err)
	}

	rowCount, err := res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}

	if rowCount != 1 {
		log.Fatalf("Expected to insert 1 row, inserted %d rows\n", rowCount)
	}
}

func (r *Repo) IncreaseStockMerce(tx *sql.Tx, merceId int64, stock int64) {
	stmt, err := tx.Prepare("UPDATE merce SET stock=stock+$2 WHERE id = $1")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(merceId, stock)
	if err != nil {
		log.Fatal(err)
	}

	rowCount, err := res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}

	if rowCount != 1 {
		log.Fatalf("Expected to update 1 row, updated %d rows\n", rowCount)
	}
}

func (r *Repo) DecrementStockMerce(tx *sql.Tx, merceId int64, stock int64) {
	stmt, err := tx.Prepare("UPDATE merce SET stock=stock-$2 WHERE id = $1")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(merceId, stock)
	if err != nil {
		log.Fatal(err)
	}

	rowCount, err := res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}

	if rowCount != 1 {
		log.Fatalf("Expected to update 1 row, updated %d rows\n", rowCount)
	}
}

func (r *Repo) InsertAddMerceStockEvent(tx *sql.Tx, addStockEvent AddStockEvent) {
	stmt, err := tx.Prepare("INSERT INTO merce_event (message) VALUES ($1)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	data, err := json.Marshal(addStockEvent)
	if err != nil {
		log.Fatal(err)
	}

	res, err := stmt.Exec(data)
	if err != nil {
		log.Fatal(err)
	}

	rowCount, err := res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}

	if rowCount != 1 {
		log.Fatalf("Expected to insert 1 row, inserted %d rows\n", rowCount)
	}
}

func (r *Repo) InsertCreateOrderEvent(tx *sql.Tx, orderEvent CreateOrderEvent) {
	stmt, err := tx.Prepare("INSERT INTO order_event (message) VALUES ($1)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	data, err := json.Marshal(orderEvent)
	if err != nil {
		log.Fatal(err)
	}

	res, err := stmt.Exec(data)
	if err != nil {
		log.Fatal(err)
	}

	rowCount, err := res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}

	if rowCount != 1 {
		log.Fatalf("Expected to insert 1 row, inserted %d rows\n", rowCount)
	}
}

func (r *Repo) Commit(tx *sql.Tx) {
	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
}

func InsertStockMerce(db *sql.DB) {
	addStockEvent := AddStockEvent{
		MerceId: 1,
		Stock:   10,
	}
	var err error

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	repo := Repo{}
	defer repo.Commit(tx)

	repo.IncreaseStockMerce(tx, addStockEvent.MerceId, addStockEvent.Stock)
	repo.InsertAddMerceStockEvent(tx, addStockEvent)
}

func InsertOrder(db *sql.DB) {
	orderEvent := CreateOrderEvent{
		Note: "test order",
		Merci: []OrderMerce{
			{
				MerceId: 1,
				Stock:   3,
			},
		},
	}

	var err error

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	repo := Repo{}
	defer repo.Commit(tx)

	orderEvent.OrderId = repo.CreateOrder(tx, orderEvent.Note)
	for _, merce := range orderEvent.Merci {
		repo.InsertOrderMerce(tx, orderEvent.OrderId, merce.MerceId, merce.Stock)
		repo.DecrementStockMerce(tx, merce.MerceId, merce.Stock)
	}
	repo.InsertCreateOrderEvent(tx, orderEvent)
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
		InsertStockMerce(db)
		InsertOrder(db)
		time.Sleep(1 * time.Second)
	}
}
