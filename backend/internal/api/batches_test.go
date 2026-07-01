package api

import (
	"math"
	"testing"
)

func TestAggregateMOE(t *testing.T) {
	agg := aggregateMOE("red_losses", "entities", []float64{2, 4, 3, 5, 1})
	if agg.Count != 5 {
		t.Errorf("Count = %d, want 5", agg.Count)
	}
	if agg.Mean != 3 {
		t.Errorf("Mean = %v, want 3", agg.Mean)
	}
	if agg.Min != 1 || agg.Max != 5 {
		t.Errorf("Min/Max = %v/%v, want 1/5", agg.Min, agg.Max)
	}
	// Sample stddev (n-1) of {1,2,3,4,5} is sqrt(2.5) ~= 1.5811.
	if math.Abs(agg.StdDev-1.5811388300841898) > 1e-9 {
		t.Errorf("StdDev = %v, want ~1.5811", agg.StdDev)
	}
}

func TestAggregateMOESingleValueHasZeroStdDev(t *testing.T) {
	agg := aggregateMOE("total_kills", "entities", []float64{7})
	if agg.Count != 1 || agg.Mean != 7 || agg.Min != 7 || agg.Max != 7 {
		t.Errorf("unexpected aggregate for single value: %+v", agg)
	}
	if agg.StdDev != 0 {
		t.Errorf("StdDev = %v, want 0 for a single sample", agg.StdDev)
	}
}

func TestAggregateMOEEmpty(t *testing.T) {
	agg := aggregateMOE("blue_losses", "entities", nil)
	if agg.Count != 0 || agg.Mean != 0 || agg.Min != 0 || agg.Max != 0 || agg.StdDev != 0 {
		t.Errorf("expected zero-value aggregate for no values, got %+v", agg)
	}
}
