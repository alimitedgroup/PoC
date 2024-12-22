package main

import (
	"database/sql"
	"github.com/alimitedgroup/palestra_poc/common"
	"log"
)

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

	addStockEvent := common.AddStockEvent{
		MerceId: merceId,
		Stock:   stock,
	}
	err = repo.InsertAddMerceStockEvent(tx, addStockEvent)
	if err != nil {
		return err
	}

	return nil
}

func InsertOrder(db *sql.DB, note string, merci []common.MerceStock) error {
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

	orderEvent := common.CreateOrderEvent{
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
