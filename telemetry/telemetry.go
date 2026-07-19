// Package telemetry provides the fan-out event bus the family's games hang
// their observers on: audio cues, HUD feeds, record keepers, and analytics
// all subscribe to the same stream of game events.
//
// The event type belongs to each game; a [Bus] is generic over it and adds
// the shared mechanics — nil-safe registration of [Subscriber] values
// (adapt plain functions with [SubscriberFunc]), an optional publish
// filter ([WithFilter]), and a bounded feed of recent events for HUD
// readouts ([Bus.Recent], sized by [WithFeedDepth], defaulting to
// [DefaultFeedDepth]). Create one with [NewBus] and deliver events with
// [Bus.Publish].
package telemetry

// Subscriber receives every event published on a Bus. E is the game's own
// event type.
type Subscriber[E any] interface {
	OnEvent(E)
}

// SubscriberFunc adapts a plain function to the Subscriber interface.
type SubscriberFunc[E any] func(E)

// OnEvent implements Subscriber by calling the function itself.
func (f SubscriberFunc[E]) OnEvent(e E) { f(e) }

// DefaultFeedDepth is how many recent events a bus keeps when no depth is
// configured.
const DefaultFeedDepth = 64

// Bus fans events out to its subscribers and keeps a bounded feed of the
// most recent ones. It is not safe for concurrent use; publish from the
// game's update loop.
type Bus[E any] struct {
	subs   []Subscriber[E]
	recent []E
	depth  int
	filter func(E) bool
}

// Option configures a Bus.
type Option[E any] func(*Bus[E])

// WithFeedDepth sets how many recent events the bus retains.
func WithFeedDepth[E any](n int) Option[E] {
	return func(b *Bus[E]) { b.depth = n }
}

// WithFilter drops events for which keep returns false before they reach
// the feed or any subscriber. Games use it to mute constant events such as
// footsteps.
func WithFilter[E any](keep func(E) bool) Option[E] {
	return func(b *Bus[E]) { b.filter = keep }
}

// NewBus returns a bus delivering to the given subscribers. Nil subscribers
// are ignored, so optional observers can be passed unconditionally.
func NewBus[E any](subs ...Subscriber[E]) *Bus[E] {
	kept := make([]Subscriber[E], 0, len(subs))
	for _, s := range subs {
		if s != nil {
			kept = append(kept, s)
		}
	}
	return &Bus[E]{subs: kept, depth: DefaultFeedDepth}
}

// Configure applies options to the bus and returns it, for chaining off
// NewBus.
func (b *Bus[E]) Configure(opts ...Option[E]) *Bus[E] {
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// Subscribe adds another subscriber. Nil is ignored.
func (b *Bus[E]) Subscribe(s Subscriber[E]) {
	if s != nil {
		b.subs = append(b.subs, s)
	}
}

// Publish records e in the recent feed and delivers it to every subscriber,
// unless the filter drops it.
func (b *Bus[E]) Publish(e E) {
	if b.filter != nil && !b.filter(e) {
		return
	}
	b.recent = append(b.recent, e)
	if b.depth > 0 && len(b.recent) > b.depth {
		b.recent = b.recent[len(b.recent)-b.depth:]
	}
	for _, s := range b.subs {
		s.OnEvent(e)
	}
}

// Recent returns the retained feed, oldest first. The slice is shared with
// the bus; treat it as read-only.
func (b *Bus[E]) Recent() []E {
	return b.recent
}
