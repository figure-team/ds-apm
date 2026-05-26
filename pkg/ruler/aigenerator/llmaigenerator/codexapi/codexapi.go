package codexapi

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
	DefaultEndpoint = "https://api.openai.com/v1/chat/completions"
	DefaultModel    = "gpt-5"
)

// Provider implements llmaigenerator.Provider via OpenAI Chat Completions API.
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
		return nil, fmt.Errorf("codexapi: apiKey must not be empty")
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

// Complete satisfies llmaigenerator.Provider. Posts a Chat Completions request
// with system + user messages and returns the first choice's text content.
func (p *Provider) Complete(ctx context.Context, system, user string) (string, error) {
	msgs := []chatMessage{}
	if system != "" {
		msgs = append(msgs, chatMessage{Role: "system", Content: system})
	}
	msgs = append(msgs, chatMessage{Role: "user", Content: user})

	body := chatRequest{
		Model:     p.model,
		Messages:  msgs,
		MaxTokens: p.maxTokens,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("codexapi: marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(raw))
	if err != nil {
		return "", fmt.Errorf("codexapi: build request: %w", err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("codexapi: http do: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("codexapi: http %d: %s", resp.StatusCode, truncate(string(respBody), 512))
	}
	var parsed chatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("codexapi: unmarshal response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("codexapi: response had no choices")
	}
	return parsed.Choices[0].Message.Content, nil
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model     string        `json:"model"`
	Messages  []chatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

// compile-time assertion: Provider must satisfy llmaigenerator.Provider.
var _ llmaigenerator.Provider = (*Provider)(nil)
