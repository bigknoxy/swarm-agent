package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.OlamaURL != "http://localhost:11434/v1" {
		t.Errorf("Expected OlamaURL 'http://localhost:11434/v1', got '%s'", cfg.OlamaURL)
	}
	if cfg.DefaultModel != "qwen2.5-coder:32b" {
		t.Errorf("Expected DefaultModel 'qwen2.5-coder:32b', got '%s'", cfg.DefaultModel)
	}
	if cfg.Timeout != 5*time.Minute {
		t.Errorf("Expected Timeout 5m, got %v", cfg.Timeout)
	}
	if cfg.MaxAttempts != 5 {
		t.Errorf("Expected MaxAttempts 5, got %d", cfg.MaxAttempts)
	}
	if !cfg.LTMEnabled {
		t.Error("Expected LTMEnabled to be true")
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configJSON := `{
  "ollama_url": "http://custom:11434/v1",
  "default_model": "qwen3.6:35b-a3b",
  "timeout": "10m",
  "max_attempts": 10,
  "workspace": "/tmp/custom-workspace",
  "ltm_enabled": false,
  "roles": {
    "architect": "qwen3.6:35b-a3b",
    "developer": "qwen2.5-coder:32b",
    "utility": "qwen3.5"
  }
}`
	os.WriteFile(configPath, []byte(configJSON), 0644)

	cfg := DefaultConfig()
	err := cfg.loadFromFile(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.OlamaURL != "http://custom:11434/v1" {
		t.Errorf("Expected custom OlamaURL, got '%s'", cfg.OlamaURL)
	}
	if cfg.DefaultModel != "qwen3.6:35b-a3b" {
		t.Errorf("Expected custom model, got '%s'", cfg.DefaultModel)
	}
	if cfg.Timeout != 10*time.Minute {
		t.Errorf("Expected Timeout 10m, got %v", cfg.Timeout)
	}
	if cfg.MaxAttempts != 10 {
		t.Errorf("Expected MaxAttempts 10, got %d", cfg.MaxAttempts)
	}
	if cfg.LTMEnabled {
		t.Error("Expected LTMEnabled to be false")
	}
}

func TestLoadFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("SWARM_MODEL", "test-model:latest")
	os.Setenv("SWARM_OLLAMA_URL", "http://env-test:11434")
	os.Setenv("SWARM_TIMEOUT", "2m")
	os.Setenv("SWARM_WORKSPACE", "/tmp/env-workspace")
	os.Setenv("SWARM_MAX_ATTEMPTS", "7")
	defer func() {
		os.Unsetenv("SWARM_MODEL")
		os.Unsetenv("SWARM_OLLAMA_URL")
		os.Unsetenv("SWARM_TIMEOUT")
		os.Unsetenv("SWARM_WORKSPACE")
		os.Unsetenv("SWARM_MAX_ATTEMPTS")
	}()

	cfg := DefaultConfig()
	cfg.loadFromEnv()

	if cfg.DefaultModel != "test-model:latest" {
		t.Errorf("Expected model from env, got '%s'", cfg.DefaultModel)
	}
	if cfg.OlamaURL != "http://env-test:11434" {
		t.Errorf("Expected URL from env, got '%s'", cfg.OlamaURL)
	}
	if cfg.Timeout != 2*time.Minute {
		t.Errorf("Expected timeout from env, got %v", cfg.Timeout)
	}
	if cfg.Workspace != "/tmp/env-workspace" {
		t.Errorf("Expected workspace from env, got '%s'", cfg.Workspace)
	}
	if cfg.MaxAttempts != 7 {
		t.Errorf("Expected max attempts from env, got %d", cfg.MaxAttempts)
	}
}

func TestLoadFromEnvInvalidMaxAttempts(t *testing.T) {
	os.Setenv("SWARM_MAX_ATTEMPTS", "not-a-number")
	defer os.Unsetenv("SWARM_MAX_ATTEMPTS")

	cfg := DefaultConfig()
	original := cfg.MaxAttempts
	cfg.loadFromEnv()

	if cfg.MaxAttempts != original {
		t.Errorf("Expected MaxAttempts to remain %d on invalid input, got %d", original, cfg.MaxAttempts)
	}
}

func TestLoadFromEnvInvalidTimeout(t *testing.T) {
	os.Setenv("SWARM_TIMEOUT", "invalid-duration")
	defer os.Unsetenv("SWARM_TIMEOUT")

	cfg := DefaultConfig()
	original := cfg.Timeout
	cfg.loadFromEnv()

	if cfg.Timeout != original {
		t.Errorf("Expected Timeout to remain %v on invalid input, got %v", original, cfg.Timeout)
	}
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"~/test/path", filepath.Join(home, "test/path")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"", ""},
	}

	for _, tt := range tests {
		result := expandHome(tt.input)
		if result != tt.expected {
			t.Errorf("expandHome(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestParseIntInvalidInput(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		hasError bool
	}{
		{"123", 123, false},
		{"abc", 0, true},
		{"12.3", 12, false}, // fmt.Sscanf stops at '.', reads 12
		{"", 0, true},
	}

	for _, tt := range tests {
		result, err := parseInt(tt.input)
		if tt.hasError && err == nil {
			t.Errorf("parseInt(%s) expected error, got nil", tt.input)
		}
		if !tt.hasError && err != nil {
			t.Errorf("parseInt(%s) unexpected error: %v", tt.input, err)
		}
		if result != tt.expected {
			t.Errorf("parseInt(%s) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}

func TestUnmarshalJSONNumericTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "numeric_timeout.json")
	// Timeout as nanoseconds (5 minutes = 300e9 ns)
	configJSON := `{
		"timeout": 300000000000
	}`
	os.WriteFile(configPath, []byte(configJSON), 0644)

	cfg := &Config{}
	err := cfg.loadFromFile(configPath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if cfg.Timeout != 5*time.Minute {
		t.Errorf("Expected 5m timeout, got %v", cfg.Timeout)
	}
}

func TestUnmarshalJSONInvalidTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid_timeout.json")
	configJSON := `{
		"timeout": "invalid-duration"
	}`
	os.WriteFile(configPath, []byte(configJSON), 0644)

	cfg := &Config{}
	err := cfg.loadFromFile(configPath)
	if err == nil {
		t.Error("Expected error for invalid timeout string")
	}
}

func TestLoadConfigNonExistentFile(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/path.json")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
	if cfg == nil {
		t.Error("Expected default config even on error")
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.json")
	os.WriteFile(configPath, []byte("{invalid json"), 0644)

	cfg, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
	if cfg == nil {
		t.Error("Expected default config even on invalid JSON")
	}
	// Verify config has defaults
	if cfg.OlamaURL != DefaultConfig().OlamaURL {
		t.Errorf("Expected default OlamaURL, got '%s'", cfg.OlamaURL)
	}
}

func TestLoadConfigMissingFields(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "partial.json")
	// Only set a few fields, rest should use defaults
	configJSON := `{
		"default_model": "custom-model:latest",
		"max_attempts": 3
	}`
	os.WriteFile(configPath, []byte(configJSON), 0644)

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Overridden fields
	if cfg.DefaultModel != "custom-model:latest" {
		t.Errorf("Expected custom model, got '%s'", cfg.DefaultModel)
	}
	if cfg.MaxAttempts != 3 {
		t.Errorf("Expected MaxAttempts 3, got %d", cfg.MaxAttempts)
	}
	// Default fields
	if cfg.OlamaURL != DefaultConfig().OlamaURL {
		t.Errorf("Expected default OlamaURL, got '%s'", cfg.OlamaURL)
	}
	if cfg.Timeout != DefaultConfig().Timeout {
		t.Errorf("Expected default Timeout, got %v", cfg.Timeout)
	}
}

func TestLoadConfigDefaultPath(t *testing.T) {
	// Test when configPath is empty (tries default location)
	// Since default config likely doesn't exist, should return defaults with no error
	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("Unexpected error for empty config path: %v", err)
	}
	if cfg.OlamaURL != DefaultConfig().OlamaURL {
		t.Errorf("Expected default OlamaURL, got '%s'", cfg.OlamaURL)
	}
}

func TestSaveToFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "save-test.json")

	cfg := DefaultConfig()
	cfg.DefaultModel = "saved-model:latest"
	cfg.Timeout = 3 * time.Minute

	err := cfg.SaveToFile(configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Reload and verify
	loaded := &Config{}
	data, _ := os.ReadFile(configPath)
	json.Unmarshal(data, loaded)

	if loaded.DefaultModel != "saved-model:latest" {
		t.Errorf("Expected saved model, got '%s'", loaded.DefaultModel)
	}
	if loaded.Timeout != 3*time.Minute {
		t.Errorf("Expected saved timeout, got %v", loaded.Timeout)
	}
}
