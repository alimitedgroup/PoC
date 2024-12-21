package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	. "magazzino/common"
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
	stmt, err := tx.Prepare("INSERT INTO merce_stock_update_event (message) VALUES ($1)")
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
	stmt, err := tx.Prepare("INSERT INTO create_order_event (message) VALUES ($1)")
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

func (r *Repo) CreateMerce(tx *sql.Tx, id int, name string, description string) error {
	_, err := tx.Exec("INSERT INTO merce (id, name, description) VALUES ($1, $2, $3)", id, name, description)
	if err != nil {
		return err
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
