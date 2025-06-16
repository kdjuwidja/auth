package main

import (
	"html/template"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kdjuwidja/aishoppercommon/db"
	"github.com/kdjuwidja/aishoppercommon/logger"
	"github.com/kdjuwidja/aishoppercommon/osutil"
	"netherealmstudio.com/m/v2/apiHandlers"
	apiHandlersaccount "netherealmstudio.com/m/v2/apiHandlers/account"
	apiHandlersauth "netherealmstudio.com/m/v2/apiHandlers/auth"
	apiHandlersdev "netherealmstudio.com/m/v2/apiHandlers/dev"
	apiHandlershealth "netherealmstudio.com/m/v2/apiHandlers/health"
	bizregister "netherealmstudio.com/m/v2/biz/register"
	dbmodel "netherealmstudio.com/m/v2/db"
	"netherealmstudio.com/m/v2/goauth"
)

func main() {
	isLocalDev := osutil.GetEnvString("IS_LOCAL_DEV", "true") == "true"

	models := []interface{}{
		&dbmodel.APIClient{},
		&dbmodel.User{},
		&dbmodel.APIClientScope{},
		&dbmodel.Role{},
		&dbmodel.RoleScope{},
		&dbmodel.UserRole{},
		&dbmodel.RegistrationCode{},
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

	// Initialize Gin router
	router := gin.Default()
	trustProxiesConf := osutil.GetEnvString("TRUST_PROXIES", "127.0.0.1")
	trustProxies := strings.Split(trustProxiesConf, ",")
	router.SetTrustedProxies(trustProxies)

	// CORS middleware
	router.Use(func(c *gin.Context) {
		corsOrigins := strings.Split(osutil.GetEnvString("CORS_ORIGINS", "http://localhost:3000"), ",")
		corsMethods := strings.Split(osutil.GetEnvString("CORS_METHODS", "GET,POST,PUT,DELETE,OPTIONS"), ",")
		corsHeaders := append(strings.Split(osutil.GetEnvString("CORS_HEADERS", "Origin,Content-Type,Accept,Authorization"), ","), "Authorization", "Content-Type")

		c.Writer.Header().Set("Access-Control-Allow-Origin", corsOrigins[0]) // For simplicity, using first origin
		c.Writer.Header().Set("Access-Control-Allow-Methods", strings.Join(corsMethods, ","))
		c.Writer.Header().Set("Access-Control-Allow-Headers", strings.Join(corsHeaders, ","))
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Serve static files
	authRouteName := osutil.GetEnvString("AUTH_ROUTE_NAME", "auth")
	accoutRouteName := osutil.GetEnvString("ACC_ROUTE_NAME", "account")
	router.Static("/"+authRouteName+"/static", "./web/static")

	// Initialize handlers
	healthHandler := apiHandlershealth.InitializeHealthHandler()
	authorizeHandler := apiHandlersauth.InitializeAuthorizeHandler(goAuth.GetSrv(), tmpl, goAuth.GetStateStore())
	tokenHandler := apiHandlersauth.InitializeTokenHandler(goAuth.GetSrv(), goAuth.GetTokenStore())
	responseFactory := apiHandlers.Initialize()
	accountHandler := apiHandlersaccount.InitializeAccountHandler(bizregister.NewRegistrationManager(mysqlConn.GetDB(), 10, osutil.GetEnvInt("USER_ROLE_ID", 2)), responseFactory)

	tokenVerifier := apiHandlers.InitializeTokenVerifier(*responseFactory)

	// Register routes for auth
	router.GET(getRoute(authRouteName, "/health"), healthHandler.HealthCheck)
	router.GET(getRoute(authRouteName, "/authorize"), authorizeHandler.Handle)
	router.POST(getRoute(authRouteName, "/authorize"), authorizeHandler.Handle)
	router.POST(getRoute(authRouteName, "/token"), tokenHandler.Handle)
	if osutil.GetEnvString("IS_LOCAL_DEV", "false") == "true" {
		tempHandler := apiHandlersdev.InitializeDevHandler()
		router.GET(getRoute(authRouteName, "/bcrypt"), tempHandler.GetBCryptHash)
	}

	// Register routes for account
	router.GET(getRoute(accoutRouteName, "/code"), tokenVerifier.VerifyToken([]string{"admin"}, accountHandler.GetRegistrationCode))
	router.POST(getRoute(accoutRouteName, "/register"), accountHandler.RegisterAccount)

	// Start server
	log.Fatal(router.Run(":9096"))
}

func getRoute(servinceName string, route string) string {
	return "/" + servinceName + route
}
