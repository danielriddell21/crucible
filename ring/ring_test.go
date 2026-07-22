package ring_test

import (
	"slices"
	"testing"

	"github.com/danielriddell21/crucible/ring"
)

func TestEmpty(t *testing.T) {
	r := ring.New[int](4)
	if r.Len() != 0 {
		t.Errorf("empty ring Len = %d, want 0", r.Len())
	}
	if r.Cap() != 4 {
		t.Errorf("Cap = %d, want 4", r.Cap())
	}
	if got := r.Slice(); len(got) != 0 {
		t.Errorf("empty ring Slice = %v, want empty", got)
	}
}

func TestFillsThenRetainsMostRecent(t *testing.T) {
	r := ring.New[int](3)
	for i := 1; i <= 5; i++ {
		r.Push(i)
	}
	if r.Len() != 3 {
		t.Errorf("Len = %d, want 3", r.Len())
	}
	if got, want := r.Slice(), []int{3, 4, 5}; !slices.Equal(got, want) {
		t.Errorf("Slice = %v, want %v (oldest first)", got, want)
	}
}

func TestPartialSliceIsOrdered(t *testing.T) {
	r := ring.New[int](5)
	r.Push(10)
	r.Push(20)
	if got, want := r.Slice(), []int{10, 20}; !slices.Equal(got, want) {
		t.Errorf("Slice = %v, want %v", got, want)
	}
}

func TestSliceIsACopy(t *testing.T) {
	r := ring.New[int](3)
	r.Push(1)
	s := r.Slice()
	s[0] = 99
	if got := r.Slice(); got[0] != 1 {
		t.Errorf("mutating a returned slice changed the ring: %v", got)
	}
}

func TestCapacityFloor(t *testing.T) {
	r := ring.New[int](0)
	r.Push(7)
	if got, want := r.Slice(), []int{7}; !slices.Equal(got, want) {
		t.Errorf("Slice = %v, want %v", got, want)
	}
}
