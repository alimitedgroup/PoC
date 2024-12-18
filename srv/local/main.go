package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	. "magazzino/common"

	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
)

type Repo struct{}

func (r *Repo) CreateOrder(tx *sql.Tx, note string) (int64, error) {
	var orderId int64
	stmt, err := tx.Prepare("INSERT INTO orders (note) VALUES ($1) RETURNING id")
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	err = stmt.QueryRow(note).Scan(&orderId)
	if err != nil {
		return 0, err
	}

	log.Printf("Inserted order with id: %d\n", orderId)
	return orderId, nil
}

func (r *Repo) InsertOrderMerce(tx *sql.Tx, orderId int64, merceId int64, stock int64) error {
	stmt, err := tx.Prepare("INSERT INTO order_merce (order_id, merce_id, stock) VALUES ($1, $2, $3)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.Exec(orderId, merceId, stock)
	if err != nil {
		return err
	}

	rowCount, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowCount != 1 {
		return fmt.Errorf("expected to insert 1 row, inserted %d rows", rowCount)
	}

	return nil
}

func (r *Repo) IncreaseStockMerce(tx *sql.Tx, merceId int64, stock int64) error {
	stmt, err := tx.Prepare("UPDATE merce SET stock=stock+$2 WHERE id = $1")
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.Exec(merceId, stock)
	if err != nil {
		return err
	}

	rowCount, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowCount != 1 {
		return fmt.Errorf("expected to update 1 row, updated %d rows", rowCount)
	}
	return nil
}

func (r *Repo) DecrementStockMerce(tx *sql.Tx, merceId int64, stock int64) error {
	stmt, err := tx.Prepare("UPDATE merce SET stock=stock-$2 WHERE id = $1")
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.Exec(merceId, stock)
	if err != nil {
		return err
	}

	rowCount, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowCount != 1 {
		return fmt.Errorf("expected to update 1 row, updated %d rows", rowCount)
	}
	return nil
}

func (r *Repo) InsertAddMerceStockEvent(tx *sql.Tx, addStockEvent AddStockEvent) error {
	stmt, err := tx.Prepare("INSERT INTO merce_event (message) VALUES ($1)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	data, err := json.Marshal(addStockEvent)
	if err != nil {
		return err
	}

	res, err := stmt.Exec(data)
	if err != nil {
		return err
	}

	rowCount, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowCount != 1 {
		return fmt.Errorf("expected to insert 1 row, inserted %d rows", rowCount)
	}

	return nil
}

func (r *Repo) InsertCreateOrderEvent(tx *sql.Tx, orderEvent CreateOrderEvent) error {
	stmt, err := tx.Prepare("INSERT INTO order_event (message) VALUES ($1)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	data, err := json.Marshal(orderEvent)
	if err != nil {
		return err
	}

	res, err := stmt.Exec(data)
	if err != nil {
		return err
	}

	rowCount, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowCount != 1 {
		return fmt.Errorf("expected to insert 1 row, inserted %d rows", rowCount)
	}

	return nil
}

func (r *Repo) FinalizeTransaction(tx *sql.Tx, err error) {
	// transaction handling, abort if err is not nil
	if err != nil {
		if e := tx.Rollback(); e != nil {
			log.Fatalf("error rolling back transaction: %v", e)
		} else {
			log.Printf("executed rollback of the transaction")
		}
	} else {
		if e := tx.Commit(); e != nil {
			log.Fatalf("error committing transaction: %v", e)
		}
	}
}

func InsertStockMerce(db *sql.DB, merceId int64, stock int64) error {
	var err error

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	repo := Repo{}
	defer func() {
		repo.FinalizeTransaction(tx, err)
	}()

	err = repo.IncreaseStockMerce(tx, merceId, stock)
	if err != nil {
		return err
	}
	log.Printf("new stock (%d items) of merceId: %v\n", stock, merceId)

	addStockEvent := AddStockEvent{
		MerceId: merceId,
		Stock:   stock,
	}
	err = repo.InsertAddMerceStockEvent(tx, addStockEvent)
	if err != nil {
		return err
	}

	return nil
}

func InsertOrder(db *sql.DB, note string, merci []MerceStock) error {
	var err error

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	repo := Repo{}
	defer func() {
		repo.FinalizeTransaction(tx, err)
	}()

	orderId, err := repo.CreateOrder(tx, note)
	if err != nil {
		return err
	}

	for _, merce := range merci {
		err = repo.InsertOrderMerce(tx, orderId, merce.MerceId, merce.Stock)
		if err != nil {
			return err
		}
		err = repo.DecrementStockMerce(tx, merce.MerceId, merce.Stock)
		if err != nil {
			return err
		}
	}

	orderEvent := CreateOrderEvent{
		OrderId: orderId,
		Note:    note,
		Merci:   merci,
	}

	err = repo.InsertCreateOrderEvent(tx, orderEvent)
	if err != nil {
		return err
	}
	return nil
}

func getRoot(w http.ResponseWriter, r *http.Request) {
	log.Printf("got / request\n")
	io.WriteString(w, "Hello!\n")
}
func getHealth(w http.ResponseWriter, r *http.Request) {
	log.Printf("got /health_check request\n")
	io.WriteString(w, "OK\n")
}

func main() {
	dbConnStr := os.Getenv("DB_URL")
	listenPort := os.Getenv("LISTEN_PORT")

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

	merci := []MerceStock{
		{
			MerceId: 1,
			Stock:   1,
		},
	}
	orderNote := "test order note"
	var newStock int64 = 10

	http.HandleFunc("/", getRoot)
	http.HandleFunc("/health", getHealth)

	go func() {
		err = http.ListenAndServe(fmt.Sprintf(":%v", listenPort), nil)
		if err != nil {
			log.Fatal(err)
		}
	}()

	for {
		for _, merce := range merci {
			err = InsertStockMerce(db, merce.MerceId, newStock)
			if err != nil {
				log.Fatal(err)
			}
		}

		err = InsertOrder(db, orderNote, merci)
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(1 * time.Second)
	}
}
