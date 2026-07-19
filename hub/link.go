package hub

// Link is the pair of channels a window uses to talk to the hub: In carries
// messages from the other windows, Out carries this window's messages to
// them. In is closed when the leader goes away, so a child window can treat
// that as its signal to terminate.
type Link[M any] struct {
	In  <-chan M
	Out chan<- M
}

// TrySend delivers m without blocking, dropping it when the channel is
// full. Window links are lossy by design: a stalled window must never stall
// the hub.
func TrySend[M any](ch chan<- M, m M) {
	select {
	case ch <- m:
	default:
	}
}
