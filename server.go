package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-oauth2/oauth2/v4/errors"
	"github.com/go-oauth2/oauth2/v4/manage"
	"github.com/go-oauth2/oauth2/v4/models"
	"github.com/go-oauth2/oauth2/v4/server"
	"github.com/go-oauth2/oauth2/v4/store"
	"github.com/rs/cors"
	"netherealmstudio.com/m/v2/db"
	"netherealmstudio.com/m/v2/statestore"
)

type Config struct {
	DBUser      string
	DBPassword  string
	DBHost      string
	DBPort      string
	DBName      string
	CORSOrigins string
	CORSMethods string
	CORSHeaders string
}

func getConfig() *Config {
	return &Config{
		DBUser:      getEnvOrDefault("USER_DB_USER", "ai_shopper_dev"),
		DBPassword:  getEnvOrDefault("USER_DB_PASSWORD", "password"),
		DBHost:      getEnvOrDefault("USER_DB_HOST", "localhost"),
		DBPort:      getEnvOrDefault("USER_DB_PORT", "3306"),
		DBName:      getEnvOrDefault("USER_DB_NAME", "oauth2"),
		CORSOrigins: getEnvOrDefault("CORS_ORIGINS", "http://localhost:3000"),
		CORSMethods: getEnvOrDefault("CORS_METHODS", "GET,POST,PUT,DELETE,OPTIONS"),
		CORSHeaders: getEnvOrDefault("CORS_HEADERS", "Origin,Content-Type,Accept,Authorization"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	config := getConfig()

	// Load templates
	tmpl := template.Must(template.ParseFiles("templates/login.html"))

	// Initialize state store
	stateStore := statestore.NewStateStore()

	// Database connection
	dbConn, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		config.DBUser, config.DBPassword, config.DBHost, config.DBPort, config.DBName))
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer dbConn.Close()

	// Test database connection
	if err := dbConn.Ping(); err != nil {
		log.Fatalf("Error pinging database: %v", err)
	}

	manager := manage.NewDefaultManager()
	// token memory store
	manager.MustTokenStorage(store.NewMemoryTokenStore())

	// client memory store
	clientStore := store.NewClientStore()
	clients, err := db.GetClients(dbConn)
	if err != nil {
		log.Fatalf("Error loading clients: %v", err)
	}

	for _, client := range clients {
		clientStore.Set(client.ID, &models.Client{
			ID:     client.ID,
			Secret: client.Secret,
			Domain: client.Domain,
		})

		log.Println("Client:", client.ID, client.Secret, client.Domain)
	}
	manager.MapClientStorage(clientStore)

	srv := server.NewDefaultServer(manager)
	srv.SetAllowGetAccessRequest(true)
	srv.SetClientInfoHandler(server.ClientFormHandler)

	srv.UserAuthorizationHandler = func(w http.ResponseWriter, r *http.Request) (userID string, err error) {
		username := r.PostFormValue("username")
		password := r.PostFormValue("password")

		fmt.Println("username:", username)
		fmt.Println("password:", password)

		return "000000", nil
	}

	srv.SetInternalErrorHandler(func(err error) (re *errors.Response) {
		log.Println("Internal Error:", err.Error())
		return
	})

	srv.SetResponseErrorHandler(func(re *errors.Response) {
		log.Println("Response Error:", re.Error.Error())
	})

	// CORS configuration
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   strings.Split(config.CORSOrigins, ","),
		AllowedMethods:   strings.Split(config.CORSMethods, ","),
		AllowedHeaders:   append(strings.Split(config.CORSHeaders, ","), "Authorization", "Content-Type"),
		AllowCredentials: true,
		ExposedHeaders:   []string{"Content-Length"},
	})

	// Create a new mux for handling routes
	mux := http.NewServeMux()

	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			clientID := r.URL.Query().Get("client_id")
			redirectURI := r.URL.Query().Get("redirect_uri")
			state := r.URL.Query().Get("state")
			responseType := r.URL.Query().Get("response_type")
			scope := func() string {
				if s := r.URL.Query().Get("scope"); s != "" {
					return s
				}
				return "profile"
			}()

			if clientID == "" || redirectURI == "" || state == "" {
				http.Error(w, "Missing client_id, redirect_uri, or state", http.StatusBadRequest)
				return
			}

			// Store the client's state
			stateStore.Add(state, clientID, redirectURI)

			// Render template with parameters
			data := struct {
				ClientID     string
				RedirectURI  string
				State        string
				ResponseType string
				Scope        string
			}{
				ClientID:     clientID,
				RedirectURI:  redirectURI,
				State:        state,
				ResponseType: responseType,
				Scope:        scope,
			}

			if err := tmpl.Execute(w, data); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Printf("Template execution error: %v", err)
				return
			}
		case "POST":
			clientID := r.FormValue("client_id")
			redirectURI := r.FormValue("redirect_uri")
			responseType := r.FormValue("response_type")
			scope := r.FormValue("scope")
			state := r.FormValue("state")

			fmt.Println("/authorize POST clientID:", clientID)
			fmt.Println("/authorize POST redirectURI:", redirectURI)
			fmt.Println("/authorize POST responseType:", responseType)
			fmt.Println("/authorize POST scope:", scope)
			fmt.Println("/authorize POST state:", state)

			// Validate state with client info
			if !stateStore.ValidateWithClientInfo(state, clientID, redirectURI) {
				http.Error(w, "Invalid state or mismatched client information", http.StatusBadRequest)
				return
			}

			if err := srv.HandleAuthorizeRequest(w, r); err != nil {
				log.Printf("Authorization error: %v", err)
			}
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		code := r.PostFormValue("code")
		state := r.PostFormValue("state")
		redirectURI := r.PostFormValue("redirect_uri")
		clientID := r.PostFormValue("client_id")

		fmt.Println("/token POST code:", code)
		fmt.Println("/token POST state:", state)
		fmt.Println("/token POST redirectURI:", redirectURI)
		fmt.Println("/token POST grant_type:", r.PostFormValue("grant_type"))
		fmt.Println("/token POST clientID:", clientID)

		if !stateStore.ValidateWithClientInfo(state, clientID, redirectURI) {
			http.Error(w, "Invalid state or mismatched redirectURI", http.StatusBadRequest)
			return
		}

		err := srv.HandleTokenRequest(w, r)
		if err == nil {
			stateStore.DeleteState(state)
		}
	})

	// Wrap the mux with CORS handler
	handler := corsHandler.Handler(mux)

	log.Fatal(http.ListenAndServe(":9096", handler))
}
