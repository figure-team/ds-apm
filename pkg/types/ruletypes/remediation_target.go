package ruletypes

import (
	"errors"
	"fmt"
	"strings"
)

// RemediationCredentialKindPrivateKey is the only credential kind v1 permits.
// The "password" kind is reserved in the schema for a future extension (design §8-Q4).
const RemediationCredentialKindPrivateKey = "private_key"

// RemediationTarget is one SSH-reachable monitored site. Operators register
// targets; propose-time resolution maps an incident's service.name to a target
// and freezes its connection parameters onto the RemediationExecution (design §3.1).
type RemediationTarget struct {
	ID    string `json:"id"`
	OrgID string `json:"orgId"`
	Name  string `json:"name"`
	Host  string `json:"host"`
	Port  int    `json:"port"`
	User  string `json:"user"`
	// SealedCredential is the secretbox-sealed private key (PEM). Never returned
	// by read APIs — write-only (design §3.1).
	SealedCredential   string   `json:"-"`
	CredentialKind     string   `json:"credentialKind"`
	HostKeyFingerprint string   `json:"hostKeyFingerprint"`
	ServiceSelectors   []string `json:"serviceSelectors"`
	CreatedAt          string   `json:"createdAt"`
	UpdatedAt          string   `json:"updatedAt"`
}

// ValidateRemediationTarget returns nil when t is well-formed, else a joined
// field-level error. Mirrors ValidateRemediationExecution style.
func ValidateRemediationTarget(t RemediationTarget) error {
	var errs []error

	if !uuidV4Pattern.MatchString(strings.TrimSpace(t.ID)) {
		errs = append(errs, fmt.Errorf("id: must be UUID v4 (got %q)", t.ID))
	}
	pilotRequireNonEmpty(&errs, "orgId", t.OrgID)
	pilotRequireNonEmpty(&errs, "name", t.Name)
	pilotRequireNonEmpty(&errs, "host", t.Host)
	pilotRequireNonEmpty(&errs, "user", t.User)
	pilotRequireNonEmpty(&errs, "hostKeyFingerprint", t.HostKeyFingerprint)
	pilotRequireNonEmpty(&errs, "sealedCredential", t.SealedCredential)

	if t.Port < 1 || t.Port > 65535 {
		errs = append(errs, fmt.Errorf("port: must be 1..65535 (got %d)", t.Port))
	}
	// v1: private_key only (design §8-Q4).
	if strings.TrimSpace(t.CredentialKind) != RemediationCredentialKindPrivateKey {
		errs = append(errs, fmt.Errorf("credentialKind: v1 supports only %q (got %q)",
			RemediationCredentialKindPrivateKey, t.CredentialKind))
	}
	if len(t.ServiceSelectors) == 0 {
		errs = append(errs, fmt.Errorf("serviceSelectors: at least one required"))
	}
	pilotAppendSecretLikeStringErrors(&errs, "name", t.Name)

	return errors.Join(errs...)
}
