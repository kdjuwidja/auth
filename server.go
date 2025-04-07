package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-oauth2/oauth2/v4/errors"
	"github.com/go-oauth2/oauth2/v4/generates"
	"github.com/go-oauth2/oauth2/v4/manage"
	oauthmodels "github.com/go-oauth2/oauth2/v4/models"
	"github.com/go-oauth2/oauth2/v4/server"
	"github.com/go-oauth2/oauth2/v4/store"
	"github.com/golang-jwt/jwt"
	"github.com/rs/cors"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"netherealmstudio.com/m/v2/models"
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
	JWTSecret   string
}

func getConfig() *Config {
	return &Config{
		DBUser:      getEnvOrDefault("USER_DB_USER", "ai_shopper_dev"),
		DBPassword:  getEnvOrDefault("USER_DB_PASSWORD", "password"),
		DBHost:      getEnvOrDefault("USER_DB_HOST", "localhost"),
		DBPort:      getEnvOrDefault("USER_DB_PORT", "3306"),
		DBName:      getEnvOrDefault("USER_DB_NAME", "ai_shopper_auth"),
		CORSOrigins: getEnvOrDefault("CORS_ORIGINS", "http://localhost:3000"),
		CORSMethods: getEnvOrDefault("CORS_METHODS", "GET,POST,PUT,DELETE,OPTIONS"),
		CORSHeaders: getEnvOrDefault("CORS_HEADERS", "Origin,Content-Type,Accept,Authorization"),
		JWTSecret:   getEnvOrDefault("JWT_SECRET", "your-secret-key"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func validateUser(dbConn *gorm.DB, email, password string) (string, error) {
	var user models.User
	result := dbConn.Where("email = ?", email).First(&user)
	if result.Error != nil {
		return "000000", result.Error
	}

	if result.RowsAffected == 0 {
		return "000000", errors.New("user not found")
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "000000", fmt.Errorf("invalid password")
	}

	fmt.Printf("ValidateUser %v successfully\n", user)
	return user.ID, nil
}

func initializeDB(config *Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		config.DBUser, config.DBPassword, config.DBHost, config.DBPort, config.DBName)

	dbConn, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	//auto migrate models
	fmt.Println("Auto migrating models")
	dbConn.AutoMigrate(&models.APIClient{}, &models.User{})
	fmt.Println("Models migrated")

	return dbConn, nil
}

func initializeAPIClientStore(dbConn *gorm.DB, isLocalDev bool) (*store.ClientStore, error) {
	clientStore := store.NewClientStore()

	result := dbConn.Find(&models.APIClient{})
	if result.Error != nil {
		return nil, fmt.Errorf("error loading clients: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		if isLocalDev {
			//create default local dev client
			defaultClientId := getEnvOrDefault("DEFAULT_CLIENT_ID", "82ce1a881b304775ad288e57e41387f3")
			defaultClientSecret := getEnvOrDefault("DEFAULT_CLIENT_SECRET", "my_secret")
			defaultClientDomain := getEnvOrDefault("DEFAULT_CLIENT_DOMAIN", "http://localhost:3000")
			defaultIsPublic := getEnvOrDefault("DEFAULT_IS_PUBLIC", "1")
			defaultDescription := getEnvOrDefault("DEFAULT_DESCRIPTION", "Default client for ai_shopper_depot")

			client := models.APIClient{
				ID:          defaultClientId,
				Secret:      defaultClientSecret,
				Domain:      defaultClientDomain,
				IsPublic:    defaultIsPublic == "1",
				Description: defaultDescription,
			}
			dbConn.Create(&client)

			clientStore.Set(client.ID, &oauthmodels.Client{
				ID:     client.ID,
				Secret: client.Secret,
				Domain: client.Domain,
			})
		} else {
			return nil, fmt.Errorf("no clients found")
		}
	} else {
		// Retrieve all clients from db and set to client store
		var clients []models.APIClient
		result.Scan(&clients)
		for _, client := range clients {
			clientStore.Set(client.ID, &oauthmodels.Client{
				ID:     client.ID,
				Secret: client.Secret,
				Domain: client.Domain,
			})

			log.Println("Client:", client.ID, client.Secret, client.Domain)
		}
	}

	return clientStore, nil
}

func createLocalDevUser(dbConn *gorm.DB) error {
	var count int64
	result := dbConn.Find(&models.User{}).Count(&count)
	if result.Error != nil {
		return fmt.Errorf("failed to access user table: %v", result.Error)
	}

	if count == 0 {
		user1 := models.User{
			ID:       "eb5dc850f1fb40a8b9b2bffd89c6a32d",
			Email:    "kdjuwidja@netherrealmstudio.com",
			Password: "$2a$10$vZU8LUTitjbU.FrFHIVkkuF7Gb6SrF3Zz0Eqq5coet/MuYEzRQ2Qm",
			IsActive: true,
		}
		user2 := models.User{
			ID:       "73064f370eda46a48a86e1fd8118be4c",
			Email:    "timmyk@netherrealmstudio.com",
			Password: "$2a$10$s5jg5gL1/2tWDOX5JcgWk.wqZR8kTxb46X3thb8JIsaD6HVYtpKGG",
			IsActive: true,
		}
		if err := dbConn.Create(&user1).Error; err != nil {
			return fmt.Errorf("failed to create user1: %v", err)
		}
		if err := dbConn.Create(&user2).Error; err != nil {
			return fmt.Errorf("failed to create user2: %v", err)
		}
	}
	return nil
}

func main() {
	isLocalDev := getEnvOrDefault("IS_LOCAL_DEV", "false") == "true"

	config := getConfig()

	// Load templates
	tmpl := template.Must(template.ParseFiles("templates/login.html"))

	// Initialize state store
	stateStore := statestore.NewStateStore()

	//initialize db connection
	dbConn, err := initializeDB(config)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	manager := manage.NewDefaultManager()

	// token memory store
	manager.MustTokenStorage(store.NewMemoryTokenStore())

	// Configure JWT token generation
	manager.MapAccessGenerate(generates.NewJWTAccessGenerate("jwt-key", []byte(config.JWTSecret), jwt.SigningMethodHS256))

	// Initialize API client store
	clientStore, err := initializeAPIClientStore(dbConn, isLocalDev)
	if err != nil {
		log.Fatalf("Failed to initialize API client store: %v", err)
	}
	manager.MapClientStorage(clientStore)

	srv := server.NewDefaultServer(manager)
	srv.SetAllowGetAccessRequest(true)
	srv.SetClientInfoHandler(server.ClientFormHandler)

	//create default local dev user
	if isLocalDev {
		if err := createLocalDevUser(dbConn); err != nil {
			log.Fatalf("Failed to create local dev users: %v", err)
		}
	}

	//initialize handlers
	srv.UserAuthorizationHandler = func(w http.ResponseWriter, r *http.Request) (userID string, err error) {
		email := r.PostFormValue("email")
		password := r.PostFormValue("password")

		fmt.Println("email:", email)
		fmt.Println("password:", password)

		return validateUser(dbConn, email, password)
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
				Error        string
			}{
				ClientID:     clientID,
				RedirectURI:  redirectURI,
				State:        state,
				ResponseType: responseType,
				Scope:        scope,
				Error:        r.URL.Query().Get("error"),
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
