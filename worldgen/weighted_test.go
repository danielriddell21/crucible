package worldgen_test

import (
	"math"
	"testing"

	"github.com/danielriddell21/crucible/worldgen"
)

func TestWeightedChoiceEmptyOrZero(t *testing.T) {
	rng := worldgen.NewRNG(1)
	if got := worldgen.WeightedChoice(rng, nil); got != -1 {
		t.Errorf("empty = %d", got)
	}
	if got := worldgen.WeightedChoice(rng, []float64{0, 0, 0}); got != -1 {
		t.Errorf("all-zero = %d", got)
	}
	if got := worldgen.WeightedChoice(rng, []float64{-1, -2}); got != -1 {
		t.Errorf("all-negative = %d", got)
	}
}

func TestWeightedChoiceSingle(t *testing.T) {
	rng := worldgen.NewRNG(1)
	for range 20 {
		if got := worldgen.WeightedChoice(rng, []float64{5}); got != 0 {
			t.Fatalf("single = %d", got)
		}
	}
}

func TestWeightedChoiceSkipsZeroWeights(t *testing.T) {
	rng := worldgen.NewRNG(2)
	weights := []float64{0, 3, 0, 0, 2, 0}
	for range 500 {
		i := worldgen.WeightedChoice(rng, weights)
		if weights[i] == 0 {
			t.Fatalf("chose zero-weight index %d", i)
		}
	}
}

func TestWeightedChoiceDistribution(t *testing.T) {
	rng := worldgen.NewRNG(42)
	weights := []float64{1, 3, 6} // expect ~10%, 30%, 60%
	const n = 200000
	var counts [3]int
	for range n {
		counts[worldgen.WeightedChoice(rng, weights)]++
	}
	total := weights[0] + weights[1] + weights[2]
	for i, w := range weights {
		want := w / total
		got := float64(counts[i]) / n
		if math.Abs(got-want) > 0.01 {
			t.Errorf("index %d: got %.3f, want %.3f", i, got, want)
		}
	}
}

func TestWeightedChoiceDeterministic(t *testing.T) {
	weights := []float64{2, 1, 4, 3}
	a, b := worldgen.NewRNG(7), worldgen.NewRNG(7)
	for range 100 {
		if worldgen.WeightedChoice(a, weights) != worldgen.WeightedChoice(b, weights) {
			t.Fatal("same seed produced different choices")
		}
	}
}
