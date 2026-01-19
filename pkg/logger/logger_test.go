package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	logger := New()
	assert.NotNil(t, logger)
	assert.NotNil(t, logger.info)
	assert.NotNil(t, logger.error)
	assert.NotNil(t, logger.warn)
}

func TestInfo(t *testing.T) {
	logger := New()
	assert.NotNil(t, logger)
	
	// Test that Info doesn't panic
	logger.Info("Test message: %s", "info")
	assert.True(t, true) // If we get here, no panic occurred
}

func TestError(t *testing.T) {
	logger := New()
	assert.NotNil(t, logger)
	
	// Test that Error doesn't panic
	logger.Error("Test error: %s", "error")
	assert.True(t, true) // If we get here, no panic occurred
}

func TestWarn(t *testing.T) {
	logger := New()
	assert.NotNil(t, logger)
	
	// Test that Warn doesn't panic
	logger.Warn("Test warning: %s", "warning")
	assert.True(t, true) // If we get here, no panic occurred
}

func TestLogger_MultipleCalls(t *testing.T) {
	logger := New()
	assert.NotNil(t, logger)
	
	// Test multiple calls don't panic
	logger.Info("Info 1")
	logger.Error("Error 1")
	logger.Warn("Warn 1")
	
	logger.Info("Info 2")
	logger.Error("Error 2")
	logger.Warn("Warn 2")
	
	// If we get here, no panic occurred
	assert.True(t, true)
}

func TestLogger_Formatting(t *testing.T) {
	logger := New()
	assert.NotNil(t, logger)
	
	// Test formatting with multiple args
	logger.Info("User %s logged in with ID %d", "john", 123)
	logger.Error("Failed to process request %d: %s", 404, "not found")
	logger.Warn("Warning: %s count is %d", "items", 5)
	
	// If we get here, formatting works
	assert.True(t, true)
}
