package ruletypes

import (
	"math"
	"testing"
)

// referenceWindow is a hand-checkable reference set:
//
//	{2,4,4,4,5,5,7,9} -> sum=40, n=8, mean=5
//	variance = (9 + 1+1+1 + 0+0 + 4 + 16) / 8 = 32/8 = 4 -> stddev=2
var referenceWindow = []float64{2, 4, 4, 4, 5, 5, 7, 9}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestComputeBaseline(t *testing.T) {
	t.Run("mean and population stddev over a window", func(t *testing.T) {
		b := ComputeBaseline(referenceWindow)
		if !almostEqual(b.Mean, 5) {
			t.Fatalf("Mean = %v, want 5", b.Mean)
		}
		if !almostEqual(b.StdDev, 2) {
			t.Fatalf("StdDev = %v, want 2", b.StdDev)
		}
	})

	t.Run("ignores NaN and Inf values", func(t *testing.T) {
		withGarbage := []float64{2, math.NaN(), 4, 4, math.Inf(1), 4, 5, 5, math.Inf(-1), 7, 9}
		b := ComputeBaseline(withGarbage)
		if !almostEqual(b.Mean, 5) {
			t.Fatalf("Mean = %v, want 5 (NaN/Inf ignored)", b.Mean)
		}
		if !almostEqual(b.StdDev, 2) {
			t.Fatalf("StdDev = %v, want 2 (NaN/Inf ignored)", b.StdDev)
		}
	})

	t.Run("no spread when fewer than two usable points", func(t *testing.T) {
		b := ComputeBaseline([]float64{42})
		if !almostEqual(b.Mean, 42) {
			t.Fatalf("Mean = %v, want 42", b.Mean)
		}
		if !almostEqual(b.StdDev, 0) {
			t.Fatalf("StdDev = %v, want 0", b.StdDev)
		}
	})

	t.Run("empty window is a zero baseline", func(t *testing.T) {
		b := ComputeBaseline(nil)
		if !almostEqual(b.Mean, 0) || !almostEqual(b.StdDev, 0) {
			t.Fatalf("ComputeBaseline(nil) = %+v, want zero baseline", b)
		}
	})
}

// TestBaseline_Deviation is the SCOPE acceptance criterion:
// normal range -> no warning; a k·σ outlier -> a deviation is produced.
func TestBaseline_Deviation(t *testing.T) {
	b := ComputeBaseline(referenceWindow) // Mean=5, StdDev=2

	t.Run("value within band -> not breached", func(t *testing.T) {
		d := b.Deviate(6, 3) // z = (6-5)/2 = 0.5
		if !almostEqual(d.ZScore, 0.5) {
			t.Fatalf("ZScore = %v, want 0.5", d.ZScore)
		}
		if d.Breached {
			t.Fatalf("Breached = true, want false for a value inside the ±3σ band")
		}
	})

	t.Run("value above band -> breached with positive deviation", func(t *testing.T) {
		d := b.Deviate(13, 3) // z = (13-5)/2 = 4.0 > 3
		if !almostEqual(d.ZScore, 4) {
			t.Fatalf("ZScore = %v, want 4", d.ZScore)
		}
		if !d.Breached {
			t.Fatalf("Breached = false, want true for a value 4σ above the mean")
		}
	})

	t.Run("value below band -> breached (two-sided)", func(t *testing.T) {
		d := b.Deviate(-3, 3) // z = (-3-5)/2 = -4.0, |z| > 3
		if !almostEqual(d.ZScore, -4) {
			t.Fatalf("ZScore = %v, want -4", d.ZScore)
		}
		if !d.Breached {
			t.Fatalf("Breached = false, want true for a value 4σ below the mean")
		}
	})

	t.Run("boundary is exclusive: |z| == k is not a breach", func(t *testing.T) {
		d := b.Deviate(11, 3) // z = (11-5)/2 = 3.0, exactly k
		if !almostEqual(d.ZScore, 3) {
			t.Fatalf("ZScore = %v, want 3", d.ZScore)
		}
		if d.Breached {
			t.Fatalf("Breached = true, want false when |z| == k (strict > semantics)")
		}
	})

	t.Run("zero spread never breaches", func(t *testing.T) {
		flat := ComputeBaseline([]float64{10, 10, 10, 10})
		d := flat.Deviate(15, 3)
		if !almostEqual(d.ZScore, 0) {
			t.Fatalf("ZScore = %v, want 0 when baseline has no spread", d.ZScore)
		}
		if d.Breached {
			t.Fatalf("Breached = true, want false when baseline has no spread")
		}
	})
}
