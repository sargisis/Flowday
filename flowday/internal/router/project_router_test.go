package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"flowday/internal/db"
	"flowday/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB initializes an in-memory SQLite database for testing
func setupTestDB() *gorm.DB {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to test database")
	}

	// Migrate schemas
	database.AutoMigrate(&models.User{}, &models.Project{}, &models.Task{})

	// Override the global DB variable
	db.DB = database

	return database
}

// createTestProjectToken duplicates logic from router_test but creates a fresh one here to avoid circular dependencies if any
func createTestProjectToken(userID uint) string {
	var jwtSecret = []byte("super-secret-key")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": float64(userID),
		"exp":     time.Now().Add(time.Hour * 1).Unix(),
	})
	tokenString, _ := token.SignedString(jwtSecret)
	return tokenString
}

func TestProjectRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Initialize DB
	testDB := setupTestDB()

	// Setup Router
	r := gin.Default()
	Setup(r)

	// Create a test user directly in DB (to respect foreign keys if enforced, and for realism)
	// Though our auth middleware just trusts the token claim, the project service uses the user_id.
	// SQLite :memory: is blank, so we just run.

	userID := uint(1)
	token := createTestProjectToken(userID)
	authHeader := "Bearer " + token

	t.Run("Create Project", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := []byte(`{"name": "New Project"}`)
		req, _ := http.NewRequest("POST", "/api/v1/projects/", bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response models.Project
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "New Project", response.Name)
		assert.Equal(t, userID, response.UserID)
		assert.NotZero(t, response.ID)
	})

	t.Run("Get Projects", func(t *testing.T) {
		// First verify we have the project created above
		// Or create a new one to be sure
		testDB.Create(&models.Project{Name: "Existing Project", UserID: userID})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/projects/", nil)
		req.Header.Set("Authorization", authHeader)

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var projects []models.Project
		json.Unmarshal(w.Body.Bytes(), &projects)

		// Should find at least the one we just created
		found := false
		for _, p := range projects {
			if p.Name == "Existing Project" {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected to find 'Existing Project'")
	})

	t.Run("Delete Project", func(t *testing.T) {
		// Create a project to delete
		p := models.Project{Name: "To Delete", UserID: userID}
		testDB.Create(&p)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/projects/%d", p.ID), nil)
		req.Header.Set("Authorization", authHeader)

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify it's gone
		var count int64
		testDB.Model(&models.Project{}).Where("id = ?", p.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("Isolate Projects by User", func(t *testing.T) {
		// Create project for User 2
		otherUserID := uint(2)
		testDB.Create(&models.Project{Name: "User 2 Project", UserID: otherUserID})

		// Request as User 1
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/projects/", nil)
		req.Header.Set("Authorization", authHeader)
		r.ServeHTTP(w, req)

		var projects []models.Project
		json.Unmarshal(w.Body.Bytes(), &projects)

		for _, p := range projects {
			assert.NotEqual(t, "User 2 Project", p.Name, "User 1 should not see User 2's projects")
		}
	})
}
