package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

type Client struct {
	ID     string
	Secret string
	Domain string
}

func GetClients(db *sql.DB) (map[string]*Client, error) {
	rows, err := db.Query("SELECT client_id, client_secret, domain FROM api_clients")
	if err != nil {
		return nil, fmt.Errorf("error querying clients: %v", err)
	}
	defer rows.Close()

	clients := make(map[string]*Client)
	for rows.Next() {
		var client Client
		if err := rows.Scan(&client.ID, &client.Secret, &client.Domain); err != nil {
			log.Printf("Error scanning client row: %v", err)
			continue
		}
		clients[client.ID] = &client
	}
	return clients, nil
}
