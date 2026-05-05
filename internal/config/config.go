package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config holds all configuration for the Swarm agent loop.
type Config struct {
	OlamaURL            string        `json:"ollama_url"`
	DefaultModel        string        `json:"default_model"`
	Timeout             time.Duration `json:"timeout"`
	MaxAttempts         int           `json:"max_attempts"`
	Workspace           string        `json:"workspace"`
	LTMEnabled          bool          `json:"ltm_enabled"`
	KnowledgeManagerPath string        `json:"knowledge_manager_path"`
	Roles               Roles         `json:"roles"`
	Tools               Tools         `json:"tools"`
	ShellTimeout        time.Duration `json:"shell_timeout"`
}

// UnmarshalJSON implements custom JSON unmarshaling to handle timeout as string or number.
func (c *Config) UnmarshalJSON(data []byte) error {
	// Alias to avoid infinite recursion
	type Alias Config
	aux := &struct {
		Timeout json.RawMessage `json:"timeout"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Parse timeout from raw JSON
	if len(aux.Timeout) > 0 {
		// Try string first (e.g., "5m", "10s")
		var timeoutStr string
		if err := json.Unmarshal(aux.Timeout, &timeoutStr); err == nil {
			d, err := time.ParseDuration(timeoutStr)
			if err != nil {
				return fmt.Errorf("invalid timeout string %q: %w", timeoutStr, err)
			}
			c.Timeout = d
		} else {
			// Try number (nanoseconds)
			var timeoutNs int64
			if err := json.Unmarshal(aux.Timeout, &timeoutNs); err != nil {
				return fmt.Errorf("invalid timeout value: %w", err)
			}
			c.Timeout = time.Duration(timeoutNs)
		}
	}
	return nil
}

// MarshalJSON implements custom JSON marshaling to write timeout as string.
func (c *Config) MarshalJSON() ([]byte, error) {
	// Alias to avoid infinite recursion
	type Alias Config
	return json.Marshal(&struct {
		Timeout string `json:"timeout"`
		*Alias
	}{
		Timeout: c.Timeout.String(),
		Alias:   (*Alias)(c),
	})
}

// Roles defines the model assignments for each role.
type Roles struct {
	Architect string `json:"architect"`
	Developer string `json:"developer"`
	Utility   string `json:"utility"`
}

// Tools defines which tools are enabled.
type Tools struct {
	PythonREPL         ToolConfig `json:"python_repl"`
	PerformanceChecker ToolConfig `json:"performance_checker"`
	FileTool           ToolConfig `json:"file_tool"`
	ShellTool          ToolConfig `json:"shell_tool"`
}

// ToolConfig defines configuration for a single tool.
type ToolConfig struct {
	Enabled bool `json:"enabled"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		OlamaURL:            "http://localhost:11434/v1",
		DefaultModel:        "qwen2.5-coder:32b",
		Timeout:             5 * time.Minute,
		MaxAttempts:         5,
		Workspace:           ".",
		LTMEnabled:          true,
		KnowledgeManagerPath: filepath.Join(home, ".swarm", "knowledge_manager.py"),
		Roles: Roles{
			Architect: "qwen3.6:35b-a3b",
			Developer: "qwen2.5-coder:32b",
			Utility:   "qwen3.5",
		},
		Tools: Tools{
			PythonREPL:         ToolConfig{Enabled: true},
			PerformanceChecker: ToolConfig{Enabled: true},
			FileTool:           ToolConfig{Enabled: true},
			ShellTool:          ToolConfig{Enabled: true},
		},
		ShellTimeout: 5 * time.Minute,
	}
}

// LoadConfig loads configuration from multiple sources with precedence:
// 1. CLI flags (highest)
// 2. Environment variables
// 3. Config file
// 4. Defaults (lowest)
func LoadConfig(configPath string) (*Config, error) {
	cfg := DefaultConfig()
	var loadErr error

	// Load from config file if it exists
	if configPath != "" {
		if err := cfg.loadFromFile(configPath); err != nil {
			loadErr = fmt.Errorf("failed to load config file: %w", err)
		}
	} else {
		// Try default location
		home, _ := os.UserHomeDir()
		defaultPath := filepath.Join(home, ".swarm", "config.json")
		if _, err := os.Stat(defaultPath); err == nil {
			if err := cfg.loadFromFile(defaultPath); err != nil {
				loadErr = fmt.Errorf("failed to load default config: %w", err)
			}
		}
	}

	// Override with environment variables
	cfg.loadFromEnv()

	return cfg, loadErr
}

// loadFromFile loads configuration from a JSON file.
func (c *Config) loadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("invalid JSON in %s: %w", path, err)
	}

	// Expand ~ in paths
	c.Workspace = expandHome(c.Workspace)
	c.KnowledgeManagerPath = expandHome(c.KnowledgeManagerPath)

	return nil
}

// loadFromEnv overrides configuration with environment variables.
func (c *Config) loadFromEnv() {
	if v := os.Getenv("SWARM_MODEL"); v != "" {
		c.DefaultModel = v
	}
	if v := os.Getenv("SWARM_OLLAMA_URL"); v != "" {
		c.OlamaURL = v
	}
	if v := os.Getenv("SWARM_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Timeout = d
		}
	}
	if v := os.Getenv("SWARM_WORKSPACE"); v != "" {
		c.Workspace = v
	}
	if v := os.Getenv("SWARM_MAX_ATTEMPTS"); v != "" {
		if n, err := parseInt(v); err == nil {
			c.MaxAttempts = n
		}
	}
	if v := os.Getenv("SWARM_SHELL_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.ShellTimeout = d
		}
	}
}

// SaveToFile saves the configuration to a JSON file.
func (c *Config) SaveToFile(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Helper functions

func expandHome(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[1:])
	}
	return path
}

func parseInt(s string) (int, error) {
	n := 0
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
