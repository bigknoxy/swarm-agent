package llm

import "context"

// Request represents a prompt sent to the LLM.
type Request struct {
	SystemPrompt string
	UserPrompt   string
}

// Response represents the answer from the LLM.
type Response struct {
	Text string
}

// Provider is the interface for different LLM backends.
type Provider interface {
	Generate(ctx context.Context, req Request) (*Response, error)
}

// Client handles communication with the LLM provider.
type Client struct {
	provider Provider
}

func NewClient(p Provider) *Client {
	return &Client{provider: p}
}

func (c *Client) Generate(ctx context.Context, req Request) (*Response, error) {
	return c.provider.Generate(ctx, req)
}
