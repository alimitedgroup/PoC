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
