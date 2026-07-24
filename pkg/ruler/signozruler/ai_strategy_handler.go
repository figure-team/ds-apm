package signozruler

import (
	"net/http"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/http/binding"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"go.uber.org/zap"
)

func (handler *handler) PreviewAIStrategy(rw http.ResponseWriter, req *http.Request) {
	orgID, err := requireOrg(req)
	if err != nil {
		render.Error(rw, err)
		return
	}

	var strategyReq ruletypes.AIStrategyRequest
	if err := binding.JSON.BindBody(req.Body, &strategyReq); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	strategy, err := handler.aiGenerator.Generate(req.Context(), strategyReq)
	if err != nil {
		render.Error(rw, errors.WrapInvalidInputf(err, errors.CodeInvalidInput, "AI strategy preview validation failed"))
		return
	}

	record, recErr := ruletypes.NewAIStrategyHistoryRecord(strategy)
	if recErr == nil {
		if upsertErr := handler.aiHistoryStore.Upsert(req.Context(), orgID, record); upsertErr != nil {
			zap.L().Warn("ai history persist failed",
				zap.String("orgId", orgID),
				zap.String("strategyId", strategy.StrategyID),
				zap.Error(upsertErr),
			) //nolint:depguard
		}
	}

	render.Success(rw, http.StatusOK, strategy)
}

func (handler *handler) GetLatestAIStrategyHistory(rw http.ResponseWriter, req *http.Request) {
	orgID, err := requireOrg(req)
	if err != nil {
		render.Error(rw, err)
		return
	}

	lookupReq := ruletypes.AIStrategyHistoryLookupRequest{
		IncidentID:       req.URL.Query().Get("incidentId"),
		AlertFingerprint: req.URL.Query().Get("alertFingerprint"),
	}
	if err := ruletypes.ValidateAIStrategyHistoryLookup(lookupReq); err != nil {
		render.Error(rw, errors.WrapInvalidInputf(err, errors.CodeInvalidInput, "AI strategy history lookup validation failed"))
		return
	}

	record, ok, err := handler.aiHistoryStore.GetLatest(req.Context(), orgID, lookupReq)
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "fetch AI strategy history"))
		return
	}
	if !ok {
		render.Error(rw, errors.NewNotFoundf(errors.CodeNotFound, "AI strategy history was not found"))
		return
	}

	render.Success(rw, http.StatusOK, record)
}
