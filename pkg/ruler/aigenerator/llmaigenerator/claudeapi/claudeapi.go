package claudeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/aigenerator/llmaigenerator"
)

const (
	DefaultEndpoint = "https://api.anthropic.com/v1/messages"
	DefaultModel    = "claude-sonnet-4-6"
	apiVersion      = "2023-06-01"
)

// Provider implements llmaigenerator.Provider via Anthropic Messages HTTP API.
type Provider struct {
	httpClient *http.Client
	endpoint   string
	apiKey     string
	model      string
	maxTokens  int
}

type Option func(*Provider)

func WithHTTPClient(c *http.Client) Option { return func(p *Provider) { p.httpClient = c } }
func WithEndpoint(u string) Option         { return func(p *Provider) { p.endpoint = u } }
func WithModel(m string) Option            { return func(p *Provider) { p.model = m } }
func WithMaxTokens(n int) Option           { return func(p *Provider) { p.maxTokens = n } }

// New constructs a Provider. apiKey is required; everything else has defaults.
func New(apiKey string, opts ...Option) (*Provider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("claudeapi: apiKey must not be empty")
	}
	p := &Provider{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		endpoint:   DefaultEndpoint,
		apiKey:     apiKey,
		model:      DefaultModel,
		maxTokens:  2048,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p, nil
}

// Complete satisfies llmaigenerator.Provider.
func (p *Provider) Complete(ctx context.Context, system, user string) (string, error) {
	body := messagesRequest{
		Model:     p.model,
		MaxTokens: p.maxTokens,
		System:    system,
		Messages:  []messageInput{{Role: "user", Content: user}},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("claudeapi: marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(raw))
	if err != nil {
		return "", fmt.Errorf("claudeapi: build request: %w", err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", apiVersion)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("claudeapi: http do: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("claudeapi: http %d: %s", resp.StatusCode, truncate(string(respBody), 512))
	}
	var parsed messagesResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("claudeapi: unmarshal response: %w", err)
	}
	var out bytes.Buffer
	for _, block := range parsed.Content {
		if block.Type == "text" {
			out.WriteString(block.Text)
		}
	}
	return out.String(), nil
}

type messageInput struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messagesRequest struct {
	Model     string         `json:"model"`
	MaxTokens int            `json:"max_tokens"`
	System    string         `json:"system,omitempty"`
	Messages  []messageInput `json:"messages"`
}

type messagesResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

// compile-time assertion: Provider must satisfy llmaigenerator.Provider.
var _ llmaigenerator.Provider = (*Provider)(nil)
