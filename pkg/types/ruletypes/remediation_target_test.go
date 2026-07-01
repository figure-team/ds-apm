package ruletypes

import "testing"

func validTarget() RemediationTarget {
	return RemediationTarget{
		ID:                 "3f2504e0-4f89-41d3-9a0c-0305e82c3301",
		OrgID:              "org-1",
		Name:               "prod-web-01",
		Host:               "10.0.0.5",
		Port:               22,
		User:               "deploy",
		SealedCredential:   "c2VhbGVk",
		CredentialKind:     RemediationCredentialKindPrivateKey,
		HostKeyFingerprint: "SHA256:abcdef",
		ServiceSelectors:   []string{"payment"},
		CreatedAt:          "2026-07-01T00:00:00Z",
		UpdatedAt:          "2026-07-01T00:00:00Z",
	}
}

func TestValidateRemediationTarget_OK(t *testing.T) {
	if err := ValidateRemediationTarget(validTarget()); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestValidateRemediationTarget_RejectsBadUUID(t *testing.T) {
	tg := validTarget()
	tg.ID = "not-a-uuid"
	if err := ValidateRemediationTarget(tg); err == nil {
		t.Fatal("expected UUID error")
	}
}

func TestValidateRemediationTarget_RejectsEmptyHostAndUser(t *testing.T) {
	tg := validTarget()
	tg.Host = ""
	tg.User = ""
	if err := ValidateRemediationTarget(tg); err == nil {
		t.Fatal("expected host/user required error")
	}
}

func TestValidateRemediationTarget_RejectsNonPrivateKeyKind(t *testing.T) {
	tg := validTarget()
	tg.CredentialKind = "password"
	if err := ValidateRemediationTarget(tg); err == nil {
		t.Fatal("v1 must reject non private_key kind")
	}
}

func TestValidateRemediationTarget_RejectsBadPort(t *testing.T) {
	tg := validTarget()
	tg.Port = 0
	if err := ValidateRemediationTarget(tg); err == nil {
		t.Fatal("expected port range error")
	}
}
