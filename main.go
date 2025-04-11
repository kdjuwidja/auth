package main

import (
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/kdjuwidja/aishoppercommon/db"
	"github.com/kdjuwidja/aishoppercommon/logger"
	"github.com/kdjuwidja/aishoppercommon/osutil"
	"github.com/rs/cors"
	"netherealmstudio.com/m/v2/apiHandlers"
	dbmodel "netherealmstudio.com/m/v2/db"
	"netherealmstudio.com/m/v2/goauth"
)

func main() {
	isLocalDev := osutil.GetEnvString("IS_LOCAL_DEV", "false") == "true"

	models := []interface{}{
		&dbmodel.APIClient{},
		&dbmodel.User{},
	}

	mysqlConn, err := db.InitializeMySQLConnectionPool(osutil.GetEnvString("USER_DB_USER", "ai_shopper_dev"),
		osutil.GetEnvString("USER_DB_PASSWORD", "password"),
		osutil.GetEnvString("USER_DB_HOST", "localhost"),
		osutil.GetEnvString("USER_DB_PORT", "3306"),
		osutil.GetEnvString("USER_DB_NAME", "ai_shopper_auth"),
		osutil.GetEnvInt("USER_DB_MAX_OPEN_CONNS", 25),
		osutil.GetEnvInt("USER_DB_MAX_IDLE_CONNS", 10),
		models,
	)
	if err != nil {
		logger.Fatalf("Failed to initialize MySQL connection pool: %v", err)
	}
	defer mysqlConn.Close()

	// Migrate database
	logger.Info("Migrating database...")
	mysqlConn.AutoMigrate()
	logger.Info("Database migrated successfully")

	// Load templates
	tmpl := template.Must(template.ParseFiles("web/templates/login.html"))

	goAuth, err := goauth.InitializeGoAuth(mysqlConn.GetDB(), isLocalDev)
	if err != nil {
		logger.Fatalf("Failed to initialize GoAuth: %v", err)
	}

	// CORS configuration
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   strings.Split(osutil.GetEnvString("CORS_ORIGINS", "http://localhost:3000"), ","),
		AllowedMethods:   strings.Split(osutil.GetEnvString("CORS_METHODS", "GET,POST,PUT,DELETE,OPTIONS"), ","),
		AllowedHeaders:   append(strings.Split(osutil.GetEnvString("CORS_HEADERS", "Origin,Content-Type,Accept,Authorization"), ","), "Authorization", "Content-Type"),
		AllowCredentials: true,
		ExposedHeaders:   []string{"Content-Length"},
	})

	// initialize handlers
	authorizeHandler := apiHandlers.InitializeAuthorizeHandler(goAuth.GetSrv(), tmpl, goAuth.GetStateStore())
	tokenHandler := apiHandlers.InitializeTokenHandler(goAuth.GetSrv(), goAuth.GetStateStore())

	// Create a new mux for handling routes
	mux := http.NewServeMux()

	// Serve static files
	fs := http.FileServer(http.Dir("./web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))
	mux.HandleFunc("/authorize", authorizeHandler.Handle)
	mux.HandleFunc("/token", tokenHandler.Handle)

	// Wrap the mux with CORS handler
	handler := corsHandler.Handler(mux)

	log.Fatal(http.ListenAndServe(":9096", handler))
}
