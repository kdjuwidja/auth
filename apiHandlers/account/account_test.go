package apiHandlersaccount

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"netherealmstudio.com/m/v2/apiHandlers"
	bizRegister "netherealmstudio.com/m/v2/biz/register"
	"netherealmstudio.com/m/v2/db"
)

func setupTestDB(t *testing.T) *gorm.DB {
	dsn := "ai_shopper_dev:password@tcp(localhost:4306)/test_db?charset=utf8mb4&parseTime=True&loc=Local"
	gormDB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	// Drop tables if they exist
	err = gormDB.Migrator().DropTable(&db.User{}, &db.RegistrationCode{})
	require.NoError(t, err)

	// Auto migrate the schema
	err = gormDB.AutoMigrate(&db.User{}, &db.RegistrationCode{})
	require.NoError(t, err)

	return gormDB
}

func setupTestRouter(t *testing.T) (*gin.Engine, *AccountHandler) {
	gormDB := setupTestDB(t)

	registrationManager := bizRegister.NewRegistrationManager(gormDB, 3)
	responseFactory := apiHandlers.Initialize()
	accountHandler := InitializeAccountHandler(registrationManager, responseFactory)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.POST("/register", accountHandler.RegisterAccount)
	router.GET("/registration-code", accountHandler.GetRegistrationCode)

	return router, accountHandler
}

func TestGetRegistrationCode(t *testing.T) {
	router, _ := setupTestRouter(t)

	req := httptest.NewRequest("GET", "/registration-code", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "code")
	assert.NotEmpty(t, response["code"])
}

func TestRegisterAccount(t *testing.T) {
	router, _ := setupTestRouter(t)

	// First get a registration code
	req := httptest.NewRequest("GET", "/registration-code", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var codeResponse map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &codeResponse)
	require.NoError(t, err)
	code := codeResponse["code"]

	// Now test registration
	registerData := map[string]string{
		"code":     code,
		"email":    "test@example.com",
		"password": "testpassword123",
	}
	jsonData, err := json.Marshal(registerData)
	require.NoError(t, err)

	req = httptest.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "message")
	assert.Equal(t, "Account registered successfully", response["message"])
}

func TestRegisterAccountInvalidInput(t *testing.T) {
	router, _ := setupTestRouter(t)

	testCases := []struct {
		name     string
		payload  map[string]string
		expected int
	}{
		{
			name: "missing code",
			payload: map[string]string{
				"email":    "test@example.com",
				"password": "testpassword123",
			},
			expected: http.StatusBadRequest,
		},
		{
			name: "missing email",
			payload: map[string]string{
				"code":     "testcode",
				"password": "testpassword123",
			},
			expected: http.StatusBadRequest,
		},
		{
			name: "missing password",
			payload: map[string]string{
				"code":  "testcode",
				"email": "test@example.com",
			},
			expected: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonData, err := json.Marshal(tc.payload)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expected, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Contains(t, response, "code")
			assert.Contains(t, response, "error")
		})
	}
}

func TestRegisterAccountWithUsedCode(t *testing.T) {
	router, _ := setupTestRouter(t)

	// First get a registration code
	req := httptest.NewRequest("GET", "/registration-code", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var codeResponse map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &codeResponse)
	require.NoError(t, err)
	code := codeResponse["code"]

	// Register first user
	registerData := map[string]string{
		"code":     code,
		"email":    "test1@example.com",
		"password": "testpassword123",
	}
	jsonData, err := json.Marshal(registerData)
	require.NoError(t, err)

	req = httptest.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Try to register second user with same code
	registerData = map[string]string{
		"code":     code,
		"email":    "test2@example.com",
		"password": "testpassword123",
	}
	jsonData, err = json.Marshal(registerData)
	require.NoError(t, err)

	req = httptest.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should get an error response
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response, "code")
	assert.Contains(t, response, "error")
}
