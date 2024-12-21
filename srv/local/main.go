package main

import (
	"database/sql"
	"log"
	"math/rand"
	"os"
	"time"

	. "magazzino/common"

	_ "github.com/jackc/pgx/stdlib"
	"github.com/nats-io/nats.go"
)

func fakeCreateOrder(db *sql.DB) {
	orderNote := "fake note"
	for {
		rows, err := db.Query("SELECT id, stock FROM merce ORDER BY RANDOM() LIMIT FLOOR(RANDOM() * 5 + 1)")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		var merci []MerceStock
		for rows.Next() {
			var merceId int64
			var stock int64
			if err := rows.Scan(&merceId, &stock); err != nil {
				log.Fatal(err)
			}
			// select % of stock for the order
			stockSelected := max(1, stock*int64(rand.Intn(10+3)+3)/100)
			if stock > 0 {
				merci = append(merci, MerceStock{MerceId: merceId, Stock: stockSelected})
			}
		}
		if err := rows.Err(); err != nil {
			log.Fatal(err)
		}

		if len(merci) > 0 {
			err := InsertOrder(db, orderNote, merci)
			if err != nil {
				log.Fatal(err)
			}
		}

		time.Sleep(time.Duration(rand.Intn(60+10)+10) * time.Second)
	}
}

func fakeMerceStockIncrease(db *sql.DB) {
	for {
		rows, err := db.Query("SELECT id, stock FROM merce ORDER BY RANDOM() LIMIT FLOOR(RANDOM() * 3 + 1)")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		var merci []MerceStock
		for rows.Next() {
			var merceId int64
			var stock int64
			if err := rows.Scan(&merceId, &stock); err != nil {
				log.Fatal(err)
			}
			stockSelected := max(1, stock*int64(rand.Intn(10+3)+3)/100)
			if stockSelected > 0 {
				merci = append(merci, MerceStock{MerceId: merceId, Stock: stockSelected})
			}
		}
		if err := rows.Err(); err != nil {
			log.Fatal(err)
		}

		for _, merce := range merci {
			err := InsertStockMerce(db, merce.MerceId, merce.Stock)
			if err != nil {
				log.Fatal(err)
			}
		}

		time.Sleep(time.Duration(rand.Intn(6+3)+3) * time.Second)
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
	go fakeCreateOrder(db)
	go fakeMerceStockIncrease(db)

	select {}
}
