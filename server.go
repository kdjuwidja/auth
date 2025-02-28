package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"

	"github.com/go-oauth2/oauth2/v4/errors"
	"github.com/go-oauth2/oauth2/v4/manage"
	"github.com/go-oauth2/oauth2/v4/models"
	"github.com/go-oauth2/oauth2/v4/server"
	"github.com/go-oauth2/oauth2/v4/store"
)

// Helper function to get environment variable with default fallback
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getAPIClientsFromDB(clientStore *store.ClientStore) {
	// Configure MySQL database connection
	dbHost := getEnvOrDefault("USER_DB_HOST", "localhost")
	dbPort := getEnvOrDefault("USER_DB_PORT", "3306")
	dbName := getEnvOrDefault("USER_DB_NAME", "user")
	dbUser := getEnvOrDefault("USER_DB_USER", "ai_shopper_dev")
	dbPass := getEnvOrDefault("USER_DB_PASSWORD", "password")

	dbConnStr := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4",
		dbUser, dbPass, dbHost, dbPort, dbName)

	log.Println("dbConnStr", dbConnStr)

	db, err := sql.Open("mysql", dbConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test the connection
	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Successfully connected to MySQL database")

	// Query to get client credentials from database
	rows, err := db.Query("SELECT client_id, client_secret, domain FROM oauth2.api_client")
	if err != nil {
		log.Fatalf("Failed to query api_client table: %v", err)
	}
	defer rows.Close()

	// Iterate through rows and populate client store
	for rows.Next() {
		var clientID, clientSecret, domain string

		err := rows.Scan(&clientID, &clientSecret, &domain)
		if err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}

		client := &models.Client{
			ID:     clientID,
			Secret: clientSecret,
			Domain: domain,
			UserID: "",
			Public: false,
		}

		// Add client to store
		clientStore.Set(clientID, client)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		log.Fatalf("Error iterating rows: %v", err)
	}
}

func main() {
	manager := manage.NewDefaultManager()
	// token memory store
	manager.MustTokenStorage(store.NewMemoryTokenStore())

	// Create client store
	clientStore := store.NewClientStore()
	getAPIClientsFromDB(clientStore)
	// Map the client store to the manager
	manager.MapClientStorage(clientStore)

	srv := server.NewDefaultServer(manager)
	srv.SetAllowGetAccessRequest(true)
	srv.SetClientInfoHandler(server.ClientFormHandler)

	srv.UserAuthorizationHandler = func(w http.ResponseWriter, r *http.Request) (userID string, err error) {
		return "000000", nil
	}

	srv.SetInternalErrorHandler(func(err error) (re *errors.Response) {
		log.Println("Internal Error:", err.Error())
		return &errors.Response{
			Error: err,
		}
	})

	srv.SetResponseErrorHandler(func(re *errors.Response) {
		log.Println("Response Error:", re.Error.Error())
	})

	http.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		err := srv.HandleAuthorizeRequest(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})

	http.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		srv.HandleTokenRequest(w, r)
	})

	log.Fatal(http.ListenAndServe(":9096", nil))
}
