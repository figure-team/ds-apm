package llmaigenerator

import (
	"context"
	"errors"
	"testing"
)

func TestClassifyError_Auth(t *testing.T) {
	cases := []string{
		"claudecli: run claude: exit status 1 (stderr: Authentication error: invalid token)",
		"codexapi: 401 Unauthorized",
		"openai api: invalid_api_key",
		"please log in to continue",
		"oauth token has expired",
		"HTTP 403 forbidden",
	}
	for _, msg := range cases {
		if got := ClassifyError(errors.New(msg)); got != ErrorKindAuth {
			t.Fatalf("expected auth for %q; got %s", msg, got)
		}
	}
}

func TestClassifyError_Timeout(t *testing.T) {
	if got := ClassifyError(context.DeadlineExceeded); got != ErrorKindTimeout {
		t.Fatalf("DeadlineExceeded should classify as timeout; got %s", got)
	}
	// Wrapped string form (we wrap errors with fmt.Errorf which loses the
	// sentinel; classify via string fallback).
	wrapped := errors.New("llmaigenerator: provider complete: context deadline exceeded")
	if got := ClassifyError(wrapped); got != ErrorKindTimeout {
		t.Fatalf("wrapped deadline string should classify as timeout; got %s", got)
	}
}

func TestClassifyError_Other(t *testing.T) {
	cases := []string{
		"claudecli: run claude: exec: no such file or directory",
		"parse: response did not contain a JSON object",
		"connection refused",
	}
	for _, msg := range cases {
		if got := ClassifyError(errors.New(msg)); got != ErrorKindOther {
			t.Fatalf("expected other for %q; got %s", msg, got)
		}
	}
}

func TestClassifyError_Nil(t *testing.T) {
	if got := ClassifyError(nil); got != ErrorKindOther {
		t.Fatalf("nil should default to other; got %s", got)
	}
}
