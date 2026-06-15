package signozruler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/http/binding"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/ruler/aigenerator/llmaigenerator"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// APIKeyPlaceholder is the sentinel value returned when an API key is set
// (so the client knows a key exists) and accepted on PUT to mean "keep the
// existing key". It is intentionally not a valid API key.
const APIKeyPlaceholder = "<unchanged>"

// aiGeneratorRebuilder is the subset of StoreAware used by the handlers so
// tests can inject a mock without depending on the concrete aigenerator type.
type aiGeneratorRebuilder interface {
	Invalidate(orgID string)
	GeneratorFor(cfg ruletypes.AIConfig) (ruletypes.AIStrategyGenerator, error)
}

// GetAIConfig handles GET /api/v2/ds/ai/config.
// Returns the current AI config for the request's org. The apiKey field is
// scrubbed in the response: replaced with APIKeyPlaceholder when a key is set,
// or left empty when no key has been stored.
func (h *handler) GetAIConfig(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}

	cfg, err := h.aiConfigStore.Get(req.Context(), orgID, h.aiCipher.DecryptFunc())
	if errors.Is(err, ruletypes.ErrAIConfigNotFound) {
		// Return a default empty config so the UI has a well-typed object.
		render.Success(rw, http.StatusOK, ruletypes.AIConfig{
			ContractVersion: ruletypes.AIConfigContractVersion,
			OrgID:           orgID,
		})
		return
	}
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "fetch AI config"))
		return
	}

	// Scrub the API key and OAuth token before returning.
	cfg.APIKey = scrubAPIKey(cfg.APIKey)
	cfg.OAuthToken = scrubAPIKey(cfg.OAuthToken)
	render.Success(rw, http.StatusOK, cfg)
}

// UpdateAIConfig handles PUT /api/v2/ds/ai/config.
// Validates and upserts the config. If APIKey is APIKeyPlaceholder the
// existing key is preserved; otherwise the incoming plaintext is stored
// (encrypted via the configured cipher).
func (h *handler) UpdateAIConfig(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}

	var incoming ruletypes.AIConfig
	if err := binding.JSON.BindBody(req.Body, &incoming); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	// Enforce the org from claims, not from the body.
	incoming.OrgID = orgID
	incoming.ContractVersion = ruletypes.AIConfigContractVersion
	if incoming.UpdatedAt == "" {
		incoming.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	if err := ruletypes.ValidateAIConfig(incoming); err != nil {
		render.Error(rw, errors.WrapInvalidInputf(err, errors.CodeInvalidInput, "AI config validation failed"))
		return
	}

	// Handle APIKeyPlaceholder: preserve existing key.
	if incoming.APIKey == APIKeyPlaceholder {
		existing, getErr := h.aiConfigStore.Get(req.Context(), orgID, h.aiCipher.DecryptFunc())
		if getErr != nil && !errors.Is(getErr, ruletypes.ErrAIConfigNotFound) {
			render.Error(rw, errors.WrapInternalf(getErr, errors.CodeInternal, "fetch existing AI config for key preservation"))
			return
		}
		if getErr == nil {
			incoming.APIKey = existing.APIKey
		} else {
			// No existing config; treat placeholder as empty key.
			incoming.APIKey = ""
		}
	}

	// Handle placeholder for OAuthToken: preserve existing token.
	if incoming.OAuthToken == APIKeyPlaceholder {
		existing, getErr := h.aiConfigStore.Get(req.Context(), orgID, h.aiCipher.DecryptFunc())
		if getErr != nil && !errors.Is(getErr, ruletypes.ErrAIConfigNotFound) {
			render.Error(rw, errors.WrapInternalf(getErr, errors.CodeInternal, "fetch existing AI config for token preservation"))
			return
		}
		if getErr == nil {
			incoming.OAuthToken = existing.OAuthToken
		} else {
			incoming.OAuthToken = ""
		}
	}

	if err := h.aiConfigStore.Upsert(req.Context(), incoming, h.aiCipher.EncryptFunc()); err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "upsert AI config"))
		return
	}

	// Invalidate cache so the next Generate picks up the new config.
	if h.aiRebuilder != nil {
		h.aiRebuilder.Invalidate(orgID)
	}

	render.Success(rw, http.StatusNoContent, nil)
}

// TestAIConfig handles POST /api/v2/ds/ai/config/test.
// Accepts a body of AIConfig (or {} to test the saved config), builds a
// throwaway AIStrategyGenerator, invokes Generate against a canned synthetic
// Payment scenario, and returns either:
//
//	{ "ok": true, "headline": "...", "model": "..." }
//	{ "ok": false, "error": "..." }
//
// Does NOT mutate stored config.
func (h *handler) TestAIConfig(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}

	// Attempt to decode body; fall back to stored config on empty body or
	// zero-value config (indicated by missing ContractVersion).
	var incoming ruletypes.AIConfig
	_ = binding.JSON.BindBody(req.Body, &incoming)
	defer req.Body.Close() //nolint:errcheck

	var testCfg ruletypes.AIConfig
	if incoming.ContractVersion == "" || incoming.Provider == "" {
		// Use the saved config.
		stored, getErr := h.aiConfigStore.Get(req.Context(), orgID, h.aiCipher.DecryptFunc())
		if errors.Is(getErr, ruletypes.ErrAIConfigNotFound) {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(rw).Encode(aiConfigTestResponse{OK: false, Error: "no AI config saved for this org"})
			return
		}
		if getErr != nil {
			render.Error(rw, errors.WrapInternalf(getErr, errors.CodeInternal, "fetch AI config for test"))
			return
		}
		testCfg = stored
	} else {
		testCfg = incoming
		testCfg.OrgID = orgID
		testCfg.ContractVersion = ruletypes.AIConfigContractVersion

		// Substitute <unchanged> sentinels with the persisted secrets so a Test
		// request that doesn't re-paste a masked secret can still exercise the
		// configured backend. Mirrors UpdateAIConfig's handling but performs a
		// single Get to cover both secrets.
		if testCfg.APIKey == APIKeyPlaceholder || testCfg.OAuthToken == APIKeyPlaceholder {
			existing, getErr := h.aiConfigStore.Get(req.Context(), orgID, h.aiCipher.DecryptFunc())
			if getErr != nil && !errors.Is(getErr, ruletypes.ErrAIConfigNotFound) {
				render.Error(rw, errors.WrapInternalf(getErr, errors.CodeInternal, "fetch existing AI config for test sentinel substitution"))
				return
			}
			if getErr == nil {
				if testCfg.APIKey == APIKeyPlaceholder {
					testCfg.APIKey = existing.APIKey
				}
				if testCfg.OAuthToken == APIKeyPlaceholder {
					testCfg.OAuthToken = existing.OAuthToken
				}
			} else {
				if testCfg.APIKey == APIKeyPlaceholder {
					testCfg.APIKey = ""
				}
				if testCfg.OAuthToken == APIKeyPlaceholder {
					testCfg.OAuthToken = ""
				}
			}
		}
	}

	// Build a throwaway generator.
	if h.aiRebuilder == nil {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(rw).Encode(aiConfigTestResponse{OK: false, Error: "AI config test not available (rebuilder not configured)"})
		return
	}
	gen, buildErr := h.aiRebuilder.GeneratorFor(testCfg)
	if buildErr != nil {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(rw).Encode(aiConfigTestResponse{
			OK:        false,
			Error:     buildErr.Error(),
			ErrorKind: string(llmaigenerator.ClassifyError(buildErr)),
		})
		return
	}

	// Invoke against a canned synthetic Payment scenario.
	syntheticReq := cannedPaymentRequest(orgID)
	strategy, genErr := gen.Generate(req.Context(), syntheticReq)
	if genErr != nil {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(rw).Encode(aiConfigTestResponse{
			OK:        false,
			Error:     genErr.Error(),
			ErrorKind: string(llmaigenerator.ClassifyError(genErr)),
		})
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(rw).Encode(aiConfigTestResponse{
		OK:       true,
		Headline: strategy.Headline,
		Model:    strategy.Audit.Model,
	})
}

// aiConfigTestResponse is the wire type for the test endpoint response.
// ErrorKind classifies a non-OK result so the UI can surface actionable
// guidance (e.g., "re-paste OAuth token") instead of the raw stderr.
type aiConfigTestResponse struct {
	OK        bool   `json:"ok"`
	Headline  string `json:"headline,omitempty"`
	Model     string `json:"model,omitempty"`
	Error     string `json:"error,omitempty"`
	ErrorKind string `json:"errorKind,omitempty"` // "auth" | "timeout" | "other"
}

// scrubAPIKey replaces a non-empty API key with the placeholder sentinel.
// An empty key is returned as-is so the client knows no key is configured.
func scrubAPIKey(apiKey string) string {
	if apiKey != "" {
		return APIKeyPlaceholder
	}
	return ""
}

// cannedPaymentRequest builds a minimal synthetic AIStrategyRequest for
// connectivity / smoke tests on the Test endpoint.
func cannedPaymentRequest(orgID string) ruletypes.AIStrategyRequest {
	// A self-contained, groundable scenario: a bound SOP plus one evidence ref,
	// so a real LLM can return a valid "ready" strategy that cites them. Without
	// these the generated draft cannot satisfy AIStrategy validation (sopId /
	// evidence / per-hypothesis citation are required), which is the whole point
	// of the connectivity test — exercise the real Render→LLM→Parse→ground path.
	return ruletypes.AIStrategyRequest{
		IncidentID:       "INC-TEST-0",
		AlertFingerprint: "fp-test-payment",
		Labels: map[string]string{
			"alertname":    "PaymentServiceHighLatency",
			"service.name": "payment-service",
			"severity":     "warning",
			"org_id":       orgID,
		},
		Annotations: map[string]string{
			"summary": "Payment service p99 latency exceeded 2s threshold",
		},
		SOPDocument: ruletypes.SOPDocument{
			SOPID:        "SOP-TEST-PAY",
			Version:      "test-1",
			Title:        "Payment latency 대응 절차 (테스트)",
			BodyMarkdown: "## 1단계\n- 결제 성공률/지연 대시보드 확인\n- 외부 PG 응답시간 점검\n## 2단계\n- 영향 채널 식별 후 우회 라우팅 검토",
		},
		EvidenceRefs: []ruletypes.AIEvidenceRef{
			{
				RefID:       "metric:latency:payment",
				Type:        "metric",
				Observation: "p99 latency 2.4s for 5 minutes",
				Confidence:  "high",
			},
		},
	}
}
