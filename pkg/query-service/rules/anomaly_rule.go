package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/querier"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/SigNoz/signoz/pkg/types/instrumentationtypes"
	"github.com/SigNoz/signoz/pkg/types/rulestatehistorytypes"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/SigNoz/signoz/pkg/units"
	"github.com/SigNoz/signoz/pkg/valuer"

	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
)

// AnomalyRule is a rule type that warns when a series deviates from its
// statistical baseline (FR-CF7.1). Unlike ThresholdRule, which compares raw
// values against a fixed target, AnomalyRule converts each series into an
// anomaly score (a z-score measured against a moving-average ± k·σ baseline,
// see ruletypes.Baseline) and then runs that score through the same threshold
// pipeline, where the target is the band width k in units of σ.
//
// v1 is intentionally simple statistics over a single evaluation window:
// the baseline is formed from the preceding datapoints and the latest
// datapoint is evaluated against it. Learned/seasonal models are a follow-up.
type AnomalyRule struct {
	*BaseRule

	querier querier.Querier
}

var _ Rule = (*AnomalyRule)(nil)

func NewAnomalyRule(
	id string,
	orgID valuer.UUID,
	p *ruletypes.PostableRule,
	querier querier.Querier,
	logger *slog.Logger,
	opts ...RuleOption,
) (*AnomalyRule, error) {
	logger.Info("creating new AnomalyRule", slog.String("rule.id", id))

	opts = append(opts, WithLogger(logger))

	baseRule, err := NewBaseRule(id, orgID, p, opts...)
	if err != nil {
		return nil, err
	}

	return &AnomalyRule{
		BaseRule: baseRule,
		querier:  querier,
	}, nil
}

func (r *AnomalyRule) Type() ruletypes.RuleType {
	return ruletypes.RuleTypeAnomaly
}

func (r *AnomalyRule) prepareQueryRange(ctx context.Context, ts time.Time) (*qbtypes.QueryRangeRequest, error) {
	r.logger.InfoContext(
		ctx, "prepare query range request v5",
		slog.Int64("ts", ts.UnixMilli()),
		slog.Int64("eval_window", r.evalWindow.Milliseconds()),
		slog.Int64("eval_delay", r.evalDelay.Milliseconds()),
	)

	startTs, endTs := r.Timestamps(ts)
	start, end := startTs.UnixMilli(), endTs.UnixMilli()

	req := &qbtypes.QueryRangeRequest{
		Start:       uint64(start),
		End:         uint64(end),
		RequestType: qbtypes.RequestTypeTimeSeries,
		CompositeQuery: qbtypes.CompositeQuery{
			Queries: make([]qbtypes.QueryEnvelope, 0),
		},
		NoCache: true,
	}
	req.CompositeQuery.Queries = make([]qbtypes.QueryEnvelope, len(r.Condition().CompositeQuery.Queries))
	copy(req.CompositeQuery.Queries, r.Condition().CompositeQuery.Queries)
	return req, nil
}

// scoreSeries converts a raw value series into a signed anomaly-score (z-score)
// series. The baseline is computed from every datapoint except the most recent,
// and only the most recent datapoint is scored against it — this is what makes
// the evaluation an early warning against recent-normal behaviour, and it
// avoids the latest spike inflating its own baseline (self-masking).
//
// The returned series carries the original labels and a single value: the
// z-score of the latest datapoint. It is nil when there are not enough
// datapoints to form a baseline (need at least two).
func (r *AnomalyRule) scoreSeries(series *qbtypes.TimeSeries) *qbtypes.TimeSeries {
	points := series.EvaluableValues()
	// Need at least one reference point plus the point being evaluated.
	if len(points) < 2 {
		return nil
	}

	reference := make([]float64, 0, len(points)-1)
	for _, p := range points[:len(points)-1] {
		reference = append(reference, p.Value)
	}
	observed := points[len(points)-1]

	baseline := ruletypes.ComputeBaseline(reference)
	z := baseline.ZScore(observed.Value)

	return &qbtypes.TimeSeries{
		Labels: series.Labels,
		Values: []*qbtypes.TimeSeriesValue{{
			Timestamp: observed.Timestamp,
			Value:     z,
		}},
	}
}

// evalSeries scores the series and runs the anomaly threshold (target = k·σ)
// over the score, returning the matching anomaly samples (empty when the latest
// datapoint stays within the band or there is not enough data).
func (r *AnomalyRule) evalSeries(series *qbtypes.TimeSeries) (ruletypes.Vector, error) {
	scored := r.scoreSeries(series)
	if scored == nil {
		return nil, nil
	}
	return r.Threshold.Eval(scored, r.Unit(), ruletypes.EvalData{
		ActiveAlerts:  r.ActiveAlertsLabelFP(),
		SendUnmatched: r.ShouldSendUnmatched(),
	})
}

func (r *AnomalyRule) buildAndRunQuery(ctx context.Context, orgID valuer.UUID, ts time.Time) (ruletypes.Vector, error) {
	params, err := r.prepareQueryRange(ctx, ts)
	if err != nil {
		return nil, err
	}

	var results []*qbtypes.TimeSeriesData

	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.CodeNamespace:    "rules",
		instrumentationtypes.CodeFunctionName: "buildAndRunQuery",
	})

	v5Result, err := r.querier.QueryRange(ctx, orgID, params)
	if err != nil {
		return nil, err
	}

	for _, item := range v5Result.Data.Results {
		if tsData, ok := item.(*qbtypes.TimeSeriesData); ok {
			results = append(results, tsData)
		} else {
			r.logger.WarnContext(ctx, "expected qbtypes.TimeSeriesData but got unexpected type", slog.String("item.type", reflect.TypeOf(item).String()))
		}
	}

	selectedQuery := r.SelectedQuery(ctx)

	var queryResult *qbtypes.TimeSeriesData
	for _, res := range results {
		if res.QueryName == selectedQuery {
			queryResult = res
			break
		}
	}

	hasData := queryResult != nil &&
		len(queryResult.Aggregations) > 0 &&
		queryResult.Aggregations[0] != nil &&
		len(queryResult.Aggregations[0].Series) > 0

	if missingDataAlert := r.HandleMissingDataAlert(ctx, ts, hasData); missingDataAlert != nil {
		return ruletypes.Vector{*missingDataAlert}, nil
	}

	var resultVector ruletypes.Vector

	if queryResult == nil || len(queryResult.Aggregations) == 0 || queryResult.Aggregations[0] == nil {
		r.logger.WarnContext(ctx, "query result is nil", slog.String("query.name", selectedQuery))
		return resultVector, nil
	}

	seriesToProcess := queryResult.Aggregations[0].Series
	if r.ShouldSkipNewGroups() {
		filteredSeries, filterErr := r.BaseRule.FilterNewSeries(ctx, ts, seriesToProcess)
		if filterErr != nil {
			r.logger.ErrorContext(ctx, "error filtering new series", errors.Attr(filterErr))
		} else {
			seriesToProcess = filteredSeries
		}
	}

	for _, series := range seriesToProcess {
		if !r.Condition().ShouldEval(series) {
			r.logger.InfoContext(
				ctx, "not enough data points to evaluate series, skipping",
				slog.Int("series.num_points", len(series.Values)),
				slog.Int("series.required_points", r.Condition().RequiredNumPoints),
			)
			continue
		}
		resultSeries, err := r.evalSeries(series)
		if err != nil {
			return nil, err
		}
		resultVector = append(resultVector, resultSeries...)
	}

	return resultVector, nil
}

func (r *AnomalyRule) Eval(ctx context.Context, ts time.Time) (int, error) {
	prevState := r.State()

	valueFormatter := units.FormatterFromUnit(r.Unit())

	res, err := r.buildAndRunQuery(ctx, r.orgID, ts)
	if err != nil {
		return 0, err
	}

	r.mtx.Lock()
	defer r.mtx.Unlock()

	resultFPs := map[uint64]struct{}{}
	alerts := make(map[uint64]*ruletypes.Alert, len(res))

	ruleReceivers := r.Threshold.GetRuleReceivers()
	ruleReceiverMap := make(map[string][]string)
	for _, value := range ruleReceivers {
		ruleReceiverMap[value.Name] = value.Channels
	}

	for _, smpl := range res {
		l := make(map[string]string, len(smpl.Metric))
		for _, lbl := range smpl.Metric {
			l[lbl.Name] = lbl.Value
		}

		value := valueFormatter.Format(smpl.V, r.Unit())
		threshold := valueFormatter.Format(smpl.Target, smpl.TargetUnit)
		r.logger.DebugContext(
			ctx, "anomaly template data for rule", slog.String("formatter.name", valueFormatter.Name()),
			slog.String("alert.value", value), slog.String("alert.threshold", threshold),
		)

		tmplData := ruletypes.AlertTemplateDataWithIncident(l, r.annotations.Map(), value, threshold)
		defs := "{{$labels := .Labels}}{{$value := .Value}}{{$threshold := .Threshold}}"

		expand := func(text string) string {
			tmpl := ruletypes.NewTemplateExpander(
				ctx,
				defs+text,
				"__alert_"+r.Name(),
				tmplData,
				nil,
			)
			result, err := tmpl.Expand()
			if err != nil {
				result = fmt.Sprintf("<error expanding template: %s>", err)
				r.logger.ErrorContext(ctx, "expanding alert template failed", errors.Attr(err), slog.Any("alert.template_data", tmplData))
			}
			return result
		}

		lb := ruletypes.NewBuilder(smpl.Metric...).Del(ruletypes.MetricNameLabel).Del(ruletypes.TemporalityLabel)
		resultLabels := ruletypes.NewBuilder(smpl.Metric...).Del(ruletypes.MetricNameLabel).Del(ruletypes.TemporalityLabel).Labels()

		for name, value := range r.labels.Map() {
			lb.Set(name, expand(value))
		}

		lb.Set(ruletypes.AlertNameLabel, r.Name())
		lb.Set(ruletypes.AlertRuleIDLabel, r.ID())
		lb.Set(ruletypes.RuleSourceLabel, r.GeneratorURL())
		// CF-11 trigger signal (design §10): anomaly alerts carry an explicit
		// marker so the code-RCA gate stays fail-closed for everything else.
		lb.Set("anomaly", "true")

		annotations := make(ruletypes.Labels, 0, len(r.annotations.Map()))
		for name, value := range r.annotations.Map() {
			annotations = append(annotations, ruletypes.Label{Name: name, Value: expand(value)})
		}
		if smpl.IsMissing {
			lb.Set(ruletypes.AlertNameLabel, "[No data] "+r.Name())
			lb.Set(ruletypes.NoDataLabel, "true")
		}

		lbs := lb.Labels()
		h := lbs.Hash()
		resultFPs[h] = struct{}{}

		if _, ok := alerts[h]; ok {
			return 0, errors.NewInternalf(errors.CodeInternal, "duplicate alert found, vector contains metrics with the same labelset after applying alert labels")
		}

		alerts[h] = &ruletypes.Alert{
			Labels:            lbs,
			QueryResultLabels: resultLabels,
			Annotations:       annotations,
			ActiveAt:          ts,
			State:             ruletypes.StatePending,
			Value:             smpl.V,
			GeneratorURL:      r.GeneratorURL(),
			Receivers:         ruleReceiverMap[lbs.Map()[ruletypes.LabelThresholdName]],
			Missing:           smpl.IsMissing,
			IsRecovering:      smpl.IsRecovering,
		}
	}

	r.logger.InfoContext(ctx, "number of anomaly alerts found", slog.Int("alert.count", len(alerts)))

	for h, a := range alerts {
		if alert, ok := r.Active[h]; ok && alert.State != ruletypes.StateInactive {
			alert.Value = a.Value
			alert.Annotations = a.Annotations
			alert.IsRecovering = a.IsRecovering
			alert.Missing = a.Missing
			if v, ok := alert.Labels.Map()[ruletypes.LabelThresholdName]; ok {
				alert.Receivers = ruleReceiverMap[v]
			}
			continue
		}

		r.Active[h] = a
	}

	itemsToAdd := []rulestatehistorytypes.RuleStateHistory{}

	for fp, a := range r.Active {
		labelsJSON, err := json.Marshal(a.QueryResultLabels)
		if err != nil {
			r.logger.ErrorContext(ctx, "error marshaling labels", errors.Attr(err), slog.Any("alert.labels", a.Labels))
		}
		if _, ok := resultFPs[fp]; !ok {
			if a.State == ruletypes.StatePending || (!a.ResolvedAt.IsZero() && ts.Sub(a.ResolvedAt) > ruletypes.ResolvedRetention) {
				delete(r.Active, fp)
			}
			if a.State != ruletypes.StateInactive {
				r.logger.DebugContext(ctx, "converting firing alert to inactive")
				a.State = ruletypes.StateInactive
				a.ResolvedAt = ts
				itemsToAdd = append(itemsToAdd, rulestatehistorytypes.RuleStateHistory{
					RuleID:       r.ID(),
					RuleName:     r.Name(),
					State:        ruletypes.StateInactive,
					StateChanged: true,
					UnixMilli:    ts.UnixMilli(),
					Labels:       rulestatehistorytypes.LabelsString(labelsJSON),
					Fingerprint:  a.QueryResultLabels.Hash(),
					Value:        a.Value,
				})
			}
			continue
		}

		if a.State == ruletypes.StatePending && ts.Sub(a.ActiveAt) >= r.holdDuration.Duration() {
			r.logger.DebugContext(ctx, "converting pending alert to firing")
			a.State = ruletypes.StateFiring
			a.FiredAt = ts
			state := ruletypes.StateFiring
			if a.Missing {
				state = ruletypes.StateNoData
			}
			itemsToAdd = append(itemsToAdd, rulestatehistorytypes.RuleStateHistory{
				RuleID:       r.ID(),
				RuleName:     r.Name(),
				State:        state,
				StateChanged: true,
				UnixMilli:    ts.UnixMilli(),
				Labels:       rulestatehistorytypes.LabelsString(labelsJSON),
				Fingerprint:  a.QueryResultLabels.Hash(),
				Value:        a.Value,
			})
		}

		changeAlertingToRecovering := a.State == ruletypes.StateFiring && a.IsRecovering
		changeRecoveringToFiring := a.State == ruletypes.StateRecovering && !a.IsRecovering && !a.Missing
		if changeAlertingToRecovering || changeRecoveringToFiring {
			state := ruletypes.StateRecovering
			if changeRecoveringToFiring {
				state = ruletypes.StateFiring
			}
			a.State = state
			r.logger.DebugContext(ctx, "converting alert state", slog.Any("alert.state", state))
			itemsToAdd = append(itemsToAdd, rulestatehistorytypes.RuleStateHistory{
				RuleID:       r.ID(),
				RuleName:     r.Name(),
				State:        state,
				StateChanged: true,
				UnixMilli:    ts.UnixMilli(),
				Labels:       rulestatehistorytypes.LabelsString(labelsJSON),
				Fingerprint:  a.QueryResultLabels.Hash(),
				Value:        a.Value,
			})
		}
	}

	currentState := r.State()

	overallStateChanged := currentState != prevState
	for idx, item := range itemsToAdd {
		item.OverallStateChanged = overallStateChanged
		item.OverallState = currentState
		itemsToAdd[idx] = item
	}

	_ = r.RecordRuleStateHistory(ctx, itemsToAdd)

	r.health = ruletypes.HealthGood
	r.lastError = err

	return len(r.Active), nil
}

func (r *AnomalyRule) String() string {
	ar := ruletypes.PostableRule{
		AlertName:         r.name,
		RuleCondition:     r.ruleCondition,
		EvalWindow:        r.evalWindow,
		Labels:            r.labels.Map(),
		Annotations:       r.annotations.Map(),
		PreferredChannels: r.preferredChannels,
	}

	byt, err := json.Marshal(ar)
	if err != nil {
		return fmt.Sprintf("error marshaling anomaly rule: %s", err.Error())
	}

	return string(byt)
}
