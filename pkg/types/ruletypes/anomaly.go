package ruletypes

import "math"

// This file implements the v1 anomaly baseline: a simple moving-average ± k·σ
// (z-score) model computed over a single window of reference values. It carries
// no learned/seasonal component — that is an explicit follow-up. Keeping the
// statistics as pure functions here lets them be unit-tested in isolation and
// reused by the anomaly rule in pkg/query-service/rules.

// Baseline is a statistical baseline computed from a window of reference values.
// Deviation is measured as a z-score against this baseline.
type Baseline struct {
	// Mean is the arithmetic mean of the reference values.
	Mean float64
	// StdDev is the population standard deviation of the reference values.
	StdDev float64
}

// Deviation describes how far an observed value lies from a baseline band.
type Deviation struct {
	// Value is the observed value being evaluated.
	Value float64
	// Baseline is the baseline the value was evaluated against.
	Baseline Baseline
	// ZScore is the signed deviation of Value from the baseline mean in units
	// of σ: (Value - Mean) / StdDev. It is 0 when the baseline has no spread.
	ZScore float64
	// K is the band half-width, in units of σ, used for the breach test.
	K float64
	// Breached is true when the value lies strictly outside the ±K·σ band,
	// i.e. |ZScore| > K. The test is two-sided.
	Breached bool
}

// ComputeBaseline computes the mean and population standard deviation over the
// given reference values. NaN and ±Inf values are ignored. With fewer than two
// usable points the standard deviation is 0 (no meaningful spread).
func ComputeBaseline(values []float64) Baseline {
	var sum float64
	var n int
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		sum += v
		n++
	}
	if n == 0 {
		return Baseline{}
	}

	mean := sum / float64(n)
	if n < 2 {
		// A single point has no spread.
		return Baseline{Mean: mean}
	}

	var sumSq float64
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		d := v - mean
		sumSq += d * d
	}
	return Baseline{Mean: mean, StdDev: math.Sqrt(sumSq / float64(n))}
}

// ZScore returns the signed deviation of v from the baseline mean in units of
// σ. When the baseline has no spread (StdDev == 0) it returns 0.
func (b Baseline) ZScore(v float64) float64 {
	if b.StdDev == 0 {
		return 0
	}
	return (v - b.Mean) / b.StdDev
}

// Deviate evaluates v against the baseline using a ±k·σ band and reports the
// resulting deviation. The breach test is two-sided: Breached == |ZScore| > k.
func (b Baseline) Deviate(v, k float64) Deviation {
	z := b.ZScore(v)
	return Deviation{
		Value:    v,
		Baseline: b,
		ZScore:   z,
		K:        k,
		Breached: math.Abs(z) > k,
	}
}
