package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"lick-scroll/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupNotificationTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestGetNotifications_Unauthorized(t *testing.T) {
	// Setup
	logger := logger.New()
	handler := &NotificationHandler{
		logger: logger,
	}

	router := setupNotificationTestRouter()
	router.GET("/notifications", handler.GetNotifications)

	// Create request without auth
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/notifications", nil)

	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"], "Unauthorized")
}

func TestGetNotifications_Success(t *testing.T) {
	// Skip test that requires Redis - simplified version for unit tests
	t.Skip("Skipping test that requires Redis mock - coverage will be improved with integration tests")
}

func TestEnableNotifications_Unauthorized(t *testing.T) {
	// Setup
	logger := logger.New()
	handler := &NotificationHandler{
		logger: logger,
	}

	router := setupNotificationTestRouter()
	router.POST("/notifications/settings/:creator_id", handler.EnableNotifications)

	// Create request without auth
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/notifications/settings/creator-123", nil)

	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestEnableNotifications_Success(t *testing.T) {
	// Skip test that requires Redis - simplified version for unit tests
	t.Skip("Skipping test that requires Redis mock - coverage will be improved with integration tests")
}

func TestDisableNotifications_Unauthorized(t *testing.T) {
	// Setup
	logger := logger.New()
	handler := &NotificationHandler{
		logger: logger,
	}

	router := setupNotificationTestRouter()
	router.DELETE("/notifications/settings/:creator_id", handler.DisableNotifications)

	// Create request without auth
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/notifications/settings/creator-123", nil)

	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDisableNotifications_Success(t *testing.T) {
	// Skip test that requires Redis - simplified version for unit tests
	t.Skip("Skipping test that requires Redis mock - coverage will be improved with integration tests")
}

func TestGetNotificationSettings_Unauthorized(t *testing.T) {
	logger := logger.New()
	handler := &NotificationHandler{
		logger: logger,
	}

	router := setupNotificationTestRouter()
	router.GET("/notifications/settings/:creator_id", handler.GetNotificationSettings)

	// Create request without auth
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/notifications/settings/creator-123", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"], "Unauthorized")
}

func TestGetNotificationSettings_Success(t *testing.T) {
	// Skip - requires Redis mock (nil redisClient causes panic)
	t.Skip("Skipping - requires Redis mock")
}

func TestGetNotificationSettings_NotFound(t *testing.T) {
	// Skip - requires Redis mock
	t.Skip("Skipping - requires Redis mock")
}

func TestProcessNotificationQueue(t *testing.T) {
	// Skip - requires queue mock
	t.Skip("Skipping - requires queue mock")
}
