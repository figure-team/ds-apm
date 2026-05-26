package claudeapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestProvider_NewRejectsEmptyAPIKey verifies that New("") returns an error.
func TestProvider_NewRejectsEmptyAPIKey(t *testing.T) {
	_, err := New("")
	require.Error(t, err)
	require.Contains(t, err.Error(), "apiKey must not be empty")
}

// TestProvider_CompleteSendsCorrectHeaders verifies the stub receives the
// three required headers: content-type, x-api-key, anthropic-version.
func TestProvider_CompleteSendsCorrectHeaders(t *testing.T) {
	const key = "test-key-123"
	var gotReq *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReq = r
		// drain body
		_, _ = io.ReadAll(r.Body)
		writeOKResponse(w, "hello")
	}))
	defer server.Close()

	p, err := New(key, WithEndpoint(server.URL))
	require.NoError(t, err)

	_, err = p.Complete(context.Background(), "sys", "user")
	require.NoError(t, err)
	require.NotNil(t, gotReq)

	require.Equal(t, "application/json", gotReq.Header.Get("content-type"))
	require.Equal(t, key, gotReq.Header.Get("x-api-key"))
	require.Equal(t, "2023-06-01", gotReq.Header.Get("anthropic-version"))
}

// TestProvider_CompleteSendsModelAndSystem decodes the request body server-side
// and asserts the Model and System fields match what was passed to New/Complete.
func TestProvider_CompleteSendsModelAndSystem(t *testing.T) {
	const wantModel = "claude-test-model"
	const wantSystem = "you are a helpful assistant"

	var gotBody messagesRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		writeOKResponse(w, "result")
	}))
	defer server.Close()

	p, err := New("key", WithEndpoint(server.URL), WithModel(wantModel))
	require.NoError(t, err)

	_, err = p.Complete(context.Background(), wantSystem, "the user prompt")
	require.NoError(t, err)

	require.Equal(t, wantModel, gotBody.Model)
	require.Equal(t, wantSystem, gotBody.System)
	require.Len(t, gotBody.Messages, 1)
	require.Equal(t, "user", gotBody.Messages[0].Role)
}

// TestProvider_CompleteConcatsTextBlocks verifies that multiple text content
// blocks from the API response are concatenated in order.
func TestProvider_CompleteConcatsTextBlocks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		resp := messagesResponse{
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{
				{Type: "text", Text: "Hello, "},
				{Type: "tool_use", Text: "ignored"},
				{Type: "text", Text: "world!"},
			},
		}
		w.Header().Set("content-type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p, err := New("key", WithEndpoint(server.URL))
	require.NoError(t, err)

	got, err := p.Complete(context.Background(), "", "hi")
	require.NoError(t, err)
	require.Equal(t, "Hello, world!", got)
}

// TestProvider_CompleteReturnsErrorOn4xx verifies that a 4xx HTTP response
// causes Complete to return an error that contains the status code and body.
func TestProvider_CompleteReturnsErrorOn4xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"bad request body"}`)
	}))
	defer server.Close()

	p, err := New("key", WithEndpoint(server.URL))
	require.NoError(t, err)

	_, err = p.Complete(context.Background(), "", "hi")
	require.Error(t, err)
	require.Contains(t, err.Error(), "400")
	require.Contains(t, err.Error(), "bad request body")
}

// TestProvider_CompleteRespectsContextCancel verifies that cancelling the
// context before the server responds causes Complete to return an error.
func TestProvider_CompleteRespectsContextCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until either client disconnects or a generous fallback fires.
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	p, err := New("key", WithEndpoint(server.URL))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err = p.Complete(ctx, "", "hi")
	elapsed := time.Since(start)

	require.Error(t, err)
	require.Less(t, elapsed, 500*time.Millisecond, "Complete should have returned quickly after context cancel")
}

// writeOKResponse writes a minimal valid Anthropic Messages response with a
// single text block containing text.
func writeOKResponse(w http.ResponseWriter, text string) {
	resp := messagesResponse{
		Content: []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{
			{Type: "text", Text: text},
		},
	}
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// TestTruncate verifies truncation behaviour.
func TestTruncate(t *testing.T) {
	require.Equal(t, "hello", truncate("hello", 10))
	require.Equal(t, "hel...", truncate("hello!", 3))
	require.True(t, strings.HasSuffix(truncate("abcdefg", 4), "..."))
}
