package telemetry_test

import (
	"testing"

	"github.com/danielriddell21/crucible/telemetry"
)

type event struct {
	kind string
	tick int
}

func collector(dst *[]event) telemetry.Subscriber[event] {
	return telemetry.SubscriberFunc[event](func(e event) { *dst = append(*dst, e) })
}

func TestPublishFansOut(t *testing.T) {
	var a, b []event
	bus := telemetry.NewBus(collector(&a), collector(&b))
	bus.Publish(event{kind: "door"})
	if len(a) != 1 || len(b) != 1 || a[0].kind != "door" {
		t.Fatalf("a = %v, b = %v", a, b)
	}
}

func TestNilSubscribersIgnored(t *testing.T) {
	var got []event
	bus := telemetry.NewBus[event](nil, collector(&got), nil)
	bus.Publish(event{kind: "ping"})
	if len(got) != 1 {
		t.Fatalf("got = %v", got)
	}
}

func TestSubscribeLater(t *testing.T) {
	bus := telemetry.NewBus[event]()
	bus.Publish(event{kind: "early"})
	var got []event
	bus.Subscribe(collector(&got))
	bus.Subscribe(nil)
	bus.Publish(event{kind: "late"})
	if len(got) != 1 || got[0].kind != "late" {
		t.Fatalf("got = %v", got)
	}
}

func TestFilterDrops(t *testing.T) {
	var got []event
	bus := telemetry.NewBus(collector(&got)).
		Configure(telemetry.WithFilter(func(e event) bool { return e.kind != "step" }))
	bus.Publish(event{kind: "step"})
	bus.Publish(event{kind: "door"})
	if len(got) != 1 || got[0].kind != "door" {
		t.Fatalf("got = %v", got)
	}
	if len(bus.Recent()) != 1 {
		t.Fatalf("filtered events must not reach the feed: %v", bus.Recent())
	}
}

func TestRecentBounded(t *testing.T) {
	bus := telemetry.NewBus[event]().Configure(telemetry.WithFeedDepth[event](3))
	for i := range 5 {
		bus.Publish(event{tick: i})
	}
	recent := bus.Recent()
	if len(recent) != 3 {
		t.Fatalf("len(Recent) = %d", len(recent))
	}
	if recent[0].tick != 2 || recent[2].tick != 4 {
		t.Fatalf("Recent = %v", recent)
	}
}

func TestDefaultDepth(t *testing.T) {
	bus := telemetry.NewBus[event]()
	for i := range telemetry.DefaultFeedDepth + 10 {
		bus.Publish(event{tick: i})
	}
	if len(bus.Recent()) != telemetry.DefaultFeedDepth {
		t.Fatalf("len(Recent) = %d", len(bus.Recent()))
	}
}
