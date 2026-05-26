package llmaigenerator

import (
	"context"
	"errors"
	"strings"
)

// ErrorKind classifies why a provider call failed so callers (handlers, UI)
// can render actionable guidance instead of a raw stderr blob.
type ErrorKind string

const (
	// ErrorKindAuth indicates the CLI / API rejected the supplied credentials.
	// Typical operator action: re-paste a fresh OAuth token or API key.
	ErrorKindAuth ErrorKind = "auth"

	// ErrorKindTimeout indicates the call exceeded the per-call deadline.
	// Typical operator action: raise the timeout setting or investigate latency.
	ErrorKindTimeout ErrorKind = "timeout"

	// ErrorKindOther covers everything else (missing binary, parse failure,
	// transport error, etc.). UI should surface the raw message.
	ErrorKindOther ErrorKind = "other"
)

// authNeedles are case-insensitive substrings that appear in stderr from
// `claude` and `codex` CLIs (and HTTP error bodies from the api transports)
// when credentials are missing, invalid, or expired. The list is intentionally
// conservative — false positives misclassify a transient 500 as an auth
// problem and prompt the operator to re-paste a working token, which is more
// annoying than letting a generic error message through. Add patterns only
// when seen in real failure logs.
var authNeedles = []string{
	"401",
	"403",
	"unauthorized",
	"unauthenticated",
	"authentication error",
	"authentication failed",
	"invalid api key",
	"invalid_api_key",
	"invalid token",
	"invalid_grant",
	"token expired",
	"expired token",
	"oauth token",
	"please log in",
	"please sign in",
	"sign in again",
}

// ClassifyError inspects err and returns the closest ErrorKind. Returns
// ErrorKindOther for nil err — callers should branch on err == nil before
// reaching here.
func ClassifyError(err error) ErrorKind {
	if err == nil {
		return ErrorKindOther
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrorKindTimeout
	}
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "context deadline exceeded") {
		return ErrorKindTimeout
	}
	for _, n := range authNeedles {
		if strings.Contains(lower, n) {
			return ErrorKindAuth
		}
	}
	return ErrorKindOther
}
