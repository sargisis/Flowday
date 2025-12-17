package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

// Helper to create a valid token for testing
func createTestToken(userID uint) string {
	// Must match the key in internal/middleware/auth.go
	var jwtSecret = []byte("super-secret-key")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": float64(userID), // JSON numbers are floats
		"exp":     time.Now().Add(time.Hour * 1).Unix(),
	})

	tokenString, _ := token.SignedString(jwtSecret)
	return tokenString
}

func TestSetup(t *testing.T) {
	// Switch to test mode so Gin doesn't spam logs
	gin.SetMode(gin.TestMode)

	setupTestDB()

	r := gin.Default()
	Setup(r)

	t.Run("Public Route - Register", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := []byte(`{"email": "test@example.com", "password": "password123"}`)
		req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response["email"])
		assert.NotEmpty(t, response["id"])
	})

	t.Run("Public Route - Login", func(t *testing.T) {
		// Log in with the user created in Register
		w := httptest.NewRecorder()
		body := []byte(`{"email": "test@example.com", "password": "password123"}`)
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response["token"])
	})

	t.Run("Protected Route - Unauthorized", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/me", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Protected Route - Authorized", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/me", nil)

		token := createTestToken(123)
		req.Header.Set("Authorization", "Bearer "+token)

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Unmarshal reads numbers as float64
		assert.Equal(t, float64(123), response["user_id"])
	})
}
