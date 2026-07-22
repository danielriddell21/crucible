package ring_test

import (
	"fmt"

	"github.com/danielriddell21/crucible/ring"
)

// A ring keeps the most recent samples of a rolling history for plotting.
func ExampleRing() {
	r := ring.New[int](3)
	for _, n := range []int{1, 2, 3, 4, 5} {
		r.Push(n)
	}
	fmt.Println(r.Slice())
	// Output: [3 4 5]
}
