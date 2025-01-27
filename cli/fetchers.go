package main

import (
	"encoding/json"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"log"
	"net/http"
	"time"
)

var client http.Client

type NewWarehousesMsg struct {
	warehouses []string
}

type NewStockMsg struct {
	stock map[string]int
}

func FetchWarehouses() tea.Msg {
	resp, err := client.Get(fmt.Sprintf("%s/warehouses", *apiGateway))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var warehouses []string
	err = json.NewDecoder(resp.Body).Decode(&warehouses)
	if err != nil {
		log.Fatal(err)
	}

	time.Sleep(1 * time.Second)

	return NewWarehousesMsg{warehouses}
}

func FetchStock(warehouse string) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.Get(fmt.Sprintf("%s/stock/%s", *apiGateway, warehouse))
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		var stock map[string]int
		err = json.NewDecoder(resp.Body).Decode(&stock)
		if err != nil {
			log.Fatal(err)
		}

		time.Sleep(1 * time.Second)

		return NewStockMsg{stock}
	}
}
