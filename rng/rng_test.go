package rng_test

import (
	"testing"

	"github.com/danielriddell21/crucible/rng"
)

func TestStreamIsDeterministic(t *testing.T) {
	a := rng.Stream(42, 1)
	b := rng.Stream(42, 1)
	for range 100 {
		if a.Uint64() != b.Uint64() {
			t.Fatal("same seed and stream must reproduce the same sequence")
		}
	}
}

func TestStreamsAreIndependent(t *testing.T) {
	a := rng.Stream(42, 1)
	b := rng.Stream(42, 2)
	same := 0
	for range 100 {
		if a.Uint64() == b.Uint64() {
			same++
		}
	}
	if same > 5 {
		t.Errorf("different streams should decorrelate, got %d/100 collisions", same)
	}
}

func TestSeedSeparatesStreams(t *testing.T) {
	if rng.Stream(1, 1).Uint64() == rng.Stream(2, 1).Uint64() {
		t.Error("different seeds on the same stream should differ")
	}
}
