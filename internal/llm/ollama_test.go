package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// MockProvider implements Provider interface for testing
type MockProvider struct {
	ResponseText string
	Err          error
	CallCount     int
}

func (m *MockProvider) Generate(ctx context.Context, req Request) (*Response, error) {
	m.CallCount++
	if m.Err != nil {
		return nil, m.Err
	}
	return &Response{Text: m.ResponseText}, nil
}

func TestClient_Generate(t *testing.T) {
	mock := &MockProvider{ResponseText: "test response"}
	client := NewClient(mock)

	req := Request{
		SystemPrompt: "You are a helper",
		UserPrompt:   "Write code",
	}

	resp, err := client.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if resp.Text != "test response" {
		t.Errorf("Expected 'test response', got '%s'", resp.Text)
	}
	if mock.CallCount != 1 {
		t.Errorf("Expected 1 call, got %d", mock.CallCount)
	}
}

func TestClient_Generate_Error(t *testing.T) {
	mock := &MockProvider{Err: context.Canceled}
	client := NewClient(mock)

	_, err := client.Generate(context.Background(), Request{})
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestOllamaProvider_Generate(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected path /chat/completions, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		// Decode request
		var req chatRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Send response
		resp := chatResponse{
			Choices: []choice{
				{Message: message{Content: "print('hello')"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := &OllamaProvider{
		BaseURL: server.URL,
		Model:   "test-model",
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	resp, err := provider.Generate(context.Background(), Request{
		SystemPrompt: "system",
		UserPrompt:   "user",
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(resp.Text, "print('hello')") {
		t.Errorf("Expected response with print('hello'), got '%s'", resp.Text)
	}
}

func TestOllamaProvider_Generate_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	provider := &OllamaProvider{
		BaseURL: server.URL,
		Model:   "test-model",
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	_, err := provider.Generate(context.Background(), Request{})
	if err == nil {
		t.Fatal("Expected error for HTTP 500")
	}
	if !strings.Contains(err.Error(), "ollama api returned status: 500") {
		t.Errorf("Expected status error, got %v", err)
	}
}

func TestOllamaProvider_Generate_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := chatResponse{Choices: []choice{}}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := &OllamaProvider{
		BaseURL: server.URL,
		Model:   "test-model",
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	_, err := provider.Generate(context.Background(), Request{})
	if err == nil {
		t.Fatal("Expected error for empty choices")
	}
	if !strings.Contains(err.Error(), "no choices") {
		t.Errorf("Expected 'no choices' error, got %v", err)
	}
}

func TestOllamaProvider_Generate_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second)
	}))
	defer server.Close()

	provider := &OllamaProvider{
		BaseURL: server.URL,
		Model:   "test-model",
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := provider.Generate(ctx, Request{})
	if err == nil {
		t.Fatal("Expected timeout error")
	}
}

func TestExtractCode_WithMarkdown(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"```python\nprint('hello')\n```", "print('hello')"},
		{"```\ncode here\n```", "code here"},
		{"Some text ```python\nprint('hi')\n``` more text", "print('hi')"},
		{"no markdown here", "no markdown here"},
		{"```python\nline1\nline2\n```", "line1\nline2"},
	}

	for _, tt := range tests {
		result := ExtractCode(tt.input)
		if result != tt.expected {
			t.Errorf("ExtractCode(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestExtractCode_NoMarkdown(t *testing.T) {
	input := "just plain text"
	result := ExtractCode(input)
	if result != "just plain text" {
		t.Errorf("Expected 'just plain text', got '%s'", result)
	}
}

func TestExtractCode_EmptyInput(t *testing.T) {
	result := ExtractCode("")
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

func TestNewOllamaProvider(t *testing.T) {
	provider := NewOllamaProvider("http://localhost:11434/v1", "qwen2.5-coder:32b")
	if provider.BaseURL != "http://localhost:11434/v1" {
		t.Errorf("Expected BaseURL 'http://localhost:11434/v1', got '%s'", provider.BaseURL)
	}
	if provider.Model != "qwen2.5-coder:32b" {
		t.Errorf("Expected Model 'qwen2.5-coder:32b', got '%s'", provider.Model)
	}
	if provider.HTTPClient.Timeout != 5*time.Minute {
		t.Errorf("Expected timeout 5m, got %v", provider.HTTPClient.Timeout)
	}
}
