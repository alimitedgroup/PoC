package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	. "magazzino/common"
)

func InitCatalog(db *sql.DB) {
	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Error beginning transaction: %v", err)
	}

	merci := []map[string]any{
		{"id": 1, "name": "Sapone", "description": "Sapone liquido per le mani"},
		{"id": 2, "name": "Shampoo", "description": "Shampoo per capelli normali"},
		{"id": 3, "name": "Carta igienica", "description": "Carta igienica a 3 veli"},
	}

	merciQueryPart := make([]string, 0, len(merci))
	merciValues := make([]any, 0, len(merci)*3)
	merciEventQueryPart := make([]string, 0, len(merci))
	merciEventValues := make([]any, 0, len(merci))
	for i, m := range merci {
		event := CreateMerceEvent{
			Id:          m["id"].(int),
			Name:        m["name"].(string),
			Description: m["description"].(string),
		}
		message, err := json.Marshal(event)
		if err != nil {
			tx.Rollback()
			log.Fatalf("Error marshalling JSON: %v", err)
		}

		merciQueryPart = append(merciQueryPart, fmt.Sprintf("($%d, $%d, $%d)", 3*i+1, 3*i+2, 3*i+3))
		merciEventQueryPart = append(merciEventQueryPart, fmt.Sprintf("($%d)", i+1))
		merciValues = append(merciValues, event.Id, event.Name, event.Description)
		merciEventValues = append(merciEventValues, message)
	}

	query := fmt.Sprintf(`
		INSERT INTO merce (id, name, description)
		VALUES %v ON CONFLICT DO NOTHING;`, strings.Join(merciQueryPart, ", "))
	// TODO: implement update on conflict, sending update_merce_event for each conflict

	queryEvent := fmt.Sprintf(`
		INSERT INTO create_merce_event (message)
		VALUES %v;`, strings.Join(merciEventQueryPart, ", "))

	_, err = tx.Exec(query, merciValues...)
	if err != nil {
		tx.Rollback()
		log.Fatalf("Error inserting merce: %v", err)
	}

	_, err = tx.Exec(queryEvent, merciEventValues...)
	if err != nil {
		tx.Rollback()
		log.Fatalf("Error inserting create_merce_event: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		log.Fatalf("Error committing transaction: %v", err)
	}

}
