package rules

import (
	"context"
	"testing"
	"time"

	cmock "github.com/srikanthccv/ClickHouse-go-mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/signoz/pkg/instrumentation/instrumentationtest"
	"github.com/SigNoz/signoz/pkg/telemetrystore"
	"github.com/SigNoz/signoz/pkg/telemetrystore/telemetrystoretest"
	"github.com/SigNoz/signoz/pkg/types/metrictypes"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

// seriesFromValues builds a metric time series from raw float values with
// monotonically increasing timestamps.
func seriesFromValues(vals []float64) *qbtypes.TimeSeries {
	ts := &qbtypes.TimeSeries{
		Labels: []*qbtypes.Label{
			{Key: telemetrytypes.TelemetryFieldKey{Name: "service"}, Value: "checkout"},
		},
	}
	base := time.Now().Add(-time.Duration(len(vals)) * time.Minute).UnixMilli()
	for i, v := range vals {
		ts.Values = append(ts.Values, &qbtypes.TimeSeriesValue{
			Timestamp: base + int64(i)*60_000,
			Value:     v,
		})
	}
	return ts
}

// anomalyPostableRule builds a postable anomaly rule whose threshold fires when
// the anomaly score (z-score) crosses k standard deviations from the baseline.
func anomalyPostableRule(k float64, op ruletypes.CompareOperator) *ruletypes.PostableRule {
	return &ruletypes.PostableRule{
		AlertName: "anomaly eval test",
		AlertType: ruletypes.AlertTypeMetric,
		RuleType:  ruletypes.RuleTypeAnomaly,
		Evaluation: &ruletypes.EvaluationEnvelope{Kind: ruletypes.RollingEvaluation, Spec: ruletypes.RollingWindow{
			EvalWindow: valuer.MustParseTextDuration("30m"),
			Frequency:  valuer.MustParseTextDuration("1m"),
		}},
		RuleCondition: &ruletypes.RuleCondition{
			CompositeQuery: &ruletypes.AlertCompositeQuery{
				QueryType: ruletypes.QueryTypeBuilder,
				Queries: []qbtypes.QueryEnvelope{{
					Type: qbtypes.QueryTypeBuilder,
					Spec: qbtypes.QueryBuilderQuery[qbtypes.MetricAggregation]{
						Name:         "A",
						StepInterval: qbtypes.Step{Duration: time.Minute},
						Aggregations: []qbtypes.MetricAggregation{{
							MetricName:       "latency",
							TimeAggregation:  metrictypes.TimeAggregationAvg,
							SpaceAggregation: metrictypes.SpaceAggregationSum,
						}},
						Signal: telemetrytypes.SignalMetrics,
					},
				}},
			},
			Target:          &k,
			CompareOperator: op,
			MatchType:       ruletypes.AtleastOnce,
			Thresholds: &ruletypes.RuleThresholdData{
				Kind: ruletypes.BasicThresholdKind,
				Spec: ruletypes.BasicRuleThresholds{{
					Name:            "anomaly",
					TargetValue:     &k,
					MatchType:       ruletypes.AtleastOnce,
					CompareOperator: op,
				}},
			},
		},
	}
}

// newAnomalyTestRule builds an anomaly rule whose threshold fires when the
// anomaly score (z-score) exceeds k standard deviations above the baseline.
func newAnomalyTestRule(t *testing.T, k float64, op ruletypes.CompareOperator) *AnomalyRule {
	t.Helper()
	logger := instrumentationtest.New().Logger()
	rule, err := NewAnomalyRule("anomaly-1", valuer.GenerateUUID(), anomalyPostableRule(k, op), nil, logger)
	require.NoError(t, err)
	return rule
}

// TestAnomalyRule_FiresOnDeviation is the SCOPE acceptance criterion: when the
// latest datapoint deviates beyond k·σ from the baseline of the preceding
// window, the rule yields a firing anomaly sample; a value within the band
// yields nothing.
func TestAnomalyRule_FiresOnDeviation(t *testing.T) {
	// baseline {2,4,4,4,5,5,7,9} -> mean=5, stddev=2 (k=3 -> band is [-1, 11])
	baseline := []float64{2, 4, 4, 4, 5, 5, 7, 9}

	t.Run("latest value within band -> no anomaly sample", func(t *testing.T) {
		rule := newAnomalyTestRule(t, 3, ruletypes.ValueIsAbove)
		series := seriesFromValues(append(append([]float64{}, baseline...), 6)) // z=0.5
		res, err := rule.evalSeries(series)
		assert.NoError(t, err)
		assert.Empty(t, res, "value inside the band must not produce an anomaly sample")
	})

	t.Run("latest value above band -> firing anomaly sample", func(t *testing.T) {
		rule := newAnomalyTestRule(t, 3, ruletypes.ValueIsAbove)
		series := seriesFromValues(append(append([]float64{}, baseline...), 13)) // z=4
		res, err := rule.evalSeries(series)
		assert.NoError(t, err)
		require.NotEmpty(t, res, "value 4σ above baseline must produce a firing anomaly sample")
		// The reported value is the anomaly score (z-score), not the raw value.
		assert.InDelta(t, 4.0, res[0].V, 1e-9, "anomaly sample value should be the z-score")
	})

	t.Run("baseline excludes the evaluated point (no self-masking)", func(t *testing.T) {
		rule := newAnomalyTestRule(t, 3, ruletypes.ValueIsAbove)
		// If the spike were included in the baseline it would inflate mean/σ and
		// mask itself; excluding it keeps the deviation detectable.
		series := seriesFromValues(append(append([]float64{}, baseline...), 13))
		res, err := rule.evalSeries(series)
		assert.NoError(t, err)
		assert.NotEmpty(t, res)
	})

	t.Run("too few points to form a baseline -> no anomaly sample", func(t *testing.T) {
		rule := newAnomalyTestRule(t, 3, ruletypes.ValueIsAbove)
		res, err := rule.evalSeries(seriesFromValues([]float64{42}))
		assert.NoError(t, err)
		assert.Empty(t, res)
	})
}

// TestAnomalyRuleStampsAnomalyLabel drives the full Eval() pipeline via a
// ClickHouse mock (mirroring the pattern used in TestThresholdRuleNoData and
// sibling threshold tests) and asserts that every produced alert carries the
// "anomaly=true" label required by the CF-11 trigger gate (design §10).
//
// Baseline: {2,4,4,4,5,5,7,9} → mean=5, σ=2 (k=3 → band [-1, 11]).
// Eval point: 13 → z-score=4 > k=3 → fires.
func TestAnomalyRuleStampsAnomalyLabel(t *testing.T) {
	// baseline {2,4,4,4,5,5,7,9} -> mean=5, stddev=2; eval point 13 -> z=4 > k=3
	baseValues := []float64{2, 4, 4, 4, 5, 5, 7, 9, 13}
	base := time.Now().Add(-time.Duration(len(baseValues)) * time.Minute)

	cols := []cmock.ColumnType{
		{Name: "ts", Type: "DateTime"},
		{Name: "value", Type: "Float64"},
	}
	rows := make([][]any, 0, len(baseValues))
	for i, v := range baseValues {
		rows = append(rows, []any{base.Add(time.Duration(i) * time.Minute), v})
	}

	telemetryStore := telemetrystoretest.New(telemetrystore.Config{}, &queryMatcherAny{})
	// 9 args: metric_name, start, end, temporality, normalized, metric_name, start2, end2, isNaN-check
	// (nil matches any actual value per cmock semantics)
	telemetryStore.Mock().
		ExpectQuery("SELECT any").
		WithArgs(nil, nil, nil, nil, nil, nil, nil, nil, nil).
		WillReturnRows(cmock.NewRows(cols, rows))

	q, mockMetadataStore := prepareQuerierForMetrics(t, telemetryStore)
	mockMetadataStore.TypeMap["latency"] = metrictypes.GaugeType

	logger := instrumentationtest.New().Logger()
	k := float64(3)
	postable := &ruletypes.PostableRule{
		AlertName: "anomaly label test",
		AlertType: ruletypes.AlertTypeMetric,
		RuleType:  ruletypes.RuleTypeAnomaly,
		Evaluation: &ruletypes.EvaluationEnvelope{Kind: ruletypes.RollingEvaluation, Spec: ruletypes.RollingWindow{
			EvalWindow: valuer.MustParseTextDuration("30m"),
			Frequency:  valuer.MustParseTextDuration("1m"),
		}},
		RuleCondition: &ruletypes.RuleCondition{
			CompositeQuery: &ruletypes.AlertCompositeQuery{
				QueryType: ruletypes.QueryTypeBuilder,
				Queries: []qbtypes.QueryEnvelope{{
					Type: qbtypes.QueryTypeBuilder,
					Spec: qbtypes.QueryBuilderQuery[qbtypes.MetricAggregation]{
						Name:         "A",
						StepInterval: qbtypes.Step{Duration: time.Minute},
						Aggregations: []qbtypes.MetricAggregation{{
							MetricName:       "latency",
							Temporality:      metrictypes.Unspecified,
							TimeAggregation:  metrictypes.TimeAggregationAvg,
							SpaceAggregation: metrictypes.SpaceAggregationSum,
						}},
						Signal: telemetrytypes.SignalMetrics,
					},
				}},
			},
			Target:          &k,
			CompareOperator: ruletypes.ValueIsAbove,
			MatchType:       ruletypes.AtleastOnce,
			Thresholds: &ruletypes.RuleThresholdData{
				Kind: ruletypes.BasicThresholdKind,
				Spec: ruletypes.BasicRuleThresholds{{
					Name:            "anomaly",
					TargetValue:     &k,
					MatchType:       ruletypes.AtleastOnce,
					CompareOperator: ruletypes.ValueIsAbove,
				}},
			},
		},
	}

	rule, err := NewAnomalyRule("anomaly-label-1", valuer.GenerateUUID(), postable, q, logger)
	require.NoError(t, err)

	alertsFound, err := rule.Eval(context.Background(), time.Now())
	require.NoError(t, err)
	require.Greater(t, alertsFound, 0, "expected at least one firing alert from the 4σ spike")

	for _, alert := range rule.Active {
		assert.Equal(t, "true", alert.Labels.Get("anomaly"),
			"every anomaly alert must carry label anomaly=true (CF-11 trigger gate)")
	}
}
