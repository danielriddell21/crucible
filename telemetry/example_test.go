package telemetry_test

import (
	"fmt"

	"github.com/danielriddell21/crucible/telemetry"
)

// ExampleBus fans game events out to an audio observer while muting the
// constant footsteps, the way the family's games wire their buses.
func ExampleBus() {
	type event struct{ Kind string }

	audio := telemetry.SubscriberFunc[event](func(e event) {
		fmt.Println("cue:", e.Kind)
	})

	bus := telemetry.NewBus(audio, nil). // nil observers are ignored
						Configure(telemetry.WithFilter(func(e event) bool { return e.Kind != "step" }))

	bus.Publish(event{Kind: "step"})
	bus.Publish(event{Kind: "door"})
	bus.Publish(event{Kind: "secret"})
	fmt.Println("feed:", len(bus.Recent()))
	// Output:
	// cue: door
	// cue: secret
	// feed: 2
}
