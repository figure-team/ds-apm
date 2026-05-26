package codexapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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

// TestProvider_CompleteSendsAuthorizationBearer verifies that the stub receives
// the Authorization header with the Bearer token.
func TestProvider_CompleteSendsAuthorizationBearer(t *testing.T) {
	const key = "test-key-123"
	var gotReq *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReq = r
		_, _ = io.ReadAll(r.Body)
		writeOKResponse(w, "hello")
	}))
	t.Cleanup(server.Close)

	p, err := New(key, WithEndpoint(server.URL))
	require.NoError(t, err)

	_, err = p.Complete(context.Background(), "sys", "user")
	require.NoError(t, err)
	require.NotNil(t, gotReq)

	require.Equal(t, "application/json", gotReq.Header.Get("content-type"))
	require.Equal(t, "Bearer "+key, gotReq.Header.Get("authorization"))
}

// TestProvider_CompleteSendsModelAndMessages decodes the request body and
// asserts the Model and Messages fields (system + user in correct order).
func TestProvider_CompleteSendsModelAndMessages(t *testing.T) {
	const wantModel = "gpt-test-model"
	const wantSystem = "you are a helpful assistant"
	const wantUser = "the user prompt"

	var gotBody chatRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		writeOKResponse(w, "result")
	}))
	t.Cleanup(server.Close)

	p, err := New("key", WithEndpoint(server.URL), WithModel(wantModel))
	require.NoError(t, err)

	_, err = p.Complete(context.Background(), wantSystem, wantUser)
	require.NoError(t, err)

	require.Equal(t, wantModel, gotBody.Model)
	require.Len(t, gotBody.Messages, 2)
	require.Equal(t, "system", gotBody.Messages[0].Role)
	require.Equal(t, wantSystem, gotBody.Messages[0].Content)
	require.Equal(t, "user", gotBody.Messages[1].Role)
	require.Equal(t, wantUser, gotBody.Messages[1].Content)
}

// TestProvider_CompleteOmitsEmptySystemMessage verifies that when system is ""
// only the user message is sent (no system message in the array).
func TestProvider_CompleteOmitsEmptySystemMessage(t *testing.T) {
	var gotBody chatRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		writeOKResponse(w, "result")
	}))
	t.Cleanup(server.Close)

	p, err := New("key", WithEndpoint(server.URL))
	require.NoError(t, err)

	_, err = p.Complete(context.Background(), "", "hello")
	require.NoError(t, err)

	require.Len(t, gotBody.Messages, 1)
	require.Equal(t, "user", gotBody.Messages[0].Role)
	require.Equal(t, "hello", gotBody.Messages[0].Content)
}

// TestProvider_CompleteReturnsFirstChoiceContent verifies that when the server
// returns multiple choices the provider returns only choice[0].
func TestProvider_CompleteReturnsFirstChoiceContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		resp := chatResponse{
			Choices: []struct {
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				}{Role: "assistant", Content: "first choice"}},
				{Message: struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				}{Role: "assistant", Content: "second choice"}},
			},
		}
		w.Header().Set("content-type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(server.Close)

	p, err := New("key", WithEndpoint(server.URL))
	require.NoError(t, err)

	got, err := p.Complete(context.Background(), "", "hi")
	require.NoError(t, err)
	require.Equal(t, "first choice", got)
}

// TestProvider_CompleteReturnsErrorOn4xx verifies that a 4xx HTTP response
// causes Complete to return an error containing the status code and body.
func TestProvider_CompleteReturnsErrorOn4xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"bad request body"}`)
	}))
	t.Cleanup(server.Close)

	p, err := New("key", WithEndpoint(server.URL))
	require.NoError(t, err)

	_, err = p.Complete(context.Background(), "", "hi")
	require.Error(t, err)
	require.Contains(t, err.Error(), "400")
	require.Contains(t, err.Error(), "bad request body")
}

// TestProvider_CompleteReturnsErrorOnEmptyChoices verifies that a response
// with an empty choices array causes Complete to return an error.
func TestProvider_CompleteReturnsErrorOnEmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		resp := chatResponse{}
		w.Header().Set("content-type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(server.Close)

	p, err := New("key", WithEndpoint(server.URL))
	require.NoError(t, err)

	_, err = p.Complete(context.Background(), "", "hi")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no choices")
}

// TestProvider_CompleteRespectsContextCancel verifies that cancelling the
// context before the server responds causes Complete to return quickly.
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

// writeOKResponse writes a minimal valid Chat Completions response with a
// single choice containing the given text.
func writeOKResponse(w http.ResponseWriter, text string) {
	resp := chatResponse{
		Choices: []struct {
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		}{
			{Message: struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}{Role: "assistant", Content: text}},
		},
	}
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// TestTruncate verifies truncation behaviour.
func TestTruncate(t *testing.T) {
	require.Equal(t, "hello", truncate("hello", 10))
	require.Equal(t, "hel...", truncate("hello!", 3))
}
