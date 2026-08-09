package netplay_test

import (
	"strings"
	"testing"
	"time"

	"github.com/danielriddell21/crucible/netplay"
)

// state and controls stand in for a game's own snapshot and input types.
type state struct {
	Tick    int
	Players []pose
	Over    bool
}

type pose struct {
	X, Y  float64
	Flags uint8
}

type controls struct {
	Throttle float64
	Steer    float64
	Fire     bool
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func hostAndClient(t *testing.T) (*netplay.Host[state, controls], *netplay.Client[state, controls]) {
	t.Helper()
	h, err := netplay.Listen[state, controls]("127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = h.Close() })

	c, err := netplay.Join[state, controls](h.Addr())
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	waitFor(t, "the host to notice the player", h.Joined)
	return h, c
}

func TestControlsReachTheHostAndStateComesBack(t *testing.T) {
	h, c := hostAndClient(t)

	c.Send(controls{Throttle: 0.75, Steer: -0.5, Fire: true})
	waitFor(t, "controls to arrive", func() bool {
		in, ok := h.Input()
		return ok && in.Throttle == 0.75
	})
	if in, _ := h.Input(); in.Steer != -0.5 || !in.Fire {
		t.Errorf("controls arrived as %+v", in)
	}

	h.Send(state{Tick: 12, Players: []pose{{X: 3, Y: 4, Flags: 1}}, Over: true})
	waitFor(t, "a snapshot to arrive", func() bool {
		_, ok := c.Snapshot()
		return ok
	})

	s, _ := c.Snapshot()
	if s.Tick != 12 || !s.Over {
		t.Errorf("snapshot arrived as %+v", s)
	}
	if len(s.Players) != 1 || s.Players[0].X != 3 || s.Players[0].Flags != 1 {
		t.Errorf("players arrived as %+v", s.Players)
	}
}

func TestBeforeAnyoneJoins(t *testing.T) {
	// A host with nobody connected must keep running: Send is a no-op, so the
	// game loop can call it unconditionally, and Input reports that there is
	// nothing to apply.
	h, err := netplay.Listen[state, controls]("127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = h.Close() }()

	if h.Joined() {
		t.Error("a host with nobody connected reports a player")
	}
	h.Send(state{Tick: 1}) // must neither panic nor block
	if _, ok := h.Input(); ok {
		t.Error("input reported available with nobody connected")
	}
	if err := h.Err(); err != nil {
		t.Errorf("Err = %v, want nil before anything has gone wrong", err)
	}
}

func TestClientBeforeItsFirstSnapshot(t *testing.T) {
	_, c := hostAndClient(t)
	if s, ok := c.Snapshot(); ok || s.Tick != 0 {
		t.Errorf("Snapshot = %+v, %v; want the zero state and false", s, ok)
	}
}

func TestHostNoticesAPlayerLeaving(t *testing.T) {
	h, err := netplay.Listen[state, controls]("127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = h.Close() }()

	c, err := netplay.Join[state, controls](h.Addr())
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	waitFor(t, "the player to connect", h.Joined)

	_ = c.Close()
	waitFor(t, "the host to notice the player has gone", func() bool { return !h.Joined() })
}

func TestInputResetsWhenThePlayerLeaves(t *testing.T) {
	// A stale input would leave the character the player was driving stuck on
	// full throttle after they dropped.
	h, err := netplay.Listen[state, controls]("127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = h.Close() }()

	c, err := netplay.Join[state, controls](h.Addr())
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	waitFor(t, "the player to connect", h.Joined)

	c.Send(controls{Throttle: 1})
	waitFor(t, "controls to arrive", func() bool {
		in, _ := h.Input()
		return in.Throttle == 1
	})

	_ = c.Close()
	waitFor(t, "the host to notice the player has gone", func() bool { return !h.Joined() })
	if in, ok := h.Input(); ok || in.Throttle != 0 {
		t.Errorf("Input = %+v, %v; want the zero controls and false", in, ok)
	}
}

func TestASecondPlayerReplacesTheFirst(t *testing.T) {
	h, first := hostAndClient(t)

	second, err := netplay.Join[state, controls](h.Addr())
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	defer func() { _ = second.Close() }()

	waitFor(t, "the second player's controls", func() bool {
		second.Send(controls{Steer: 0.25})
		in, ok := h.Input()
		return ok && in.Steer == 0.25
	})

	// The first connection is closed, so its reader stops.
	waitFor(t, "the first player to be dropped", func() bool {
		first.Send(controls{Steer: -1})
		in, _ := h.Input()
		return in.Steer != -1
	})
}

func TestJoinFillsInTheDefaultPort(t *testing.T) {
	// Nothing is listening, so this must fail — but on the default port, which
	// is what the error names.
	_, err := netplay.Join[state, controls]("127.0.0.1")
	if err == nil {
		t.Fatal("joining a host with nothing listening succeeded")
	}
	if !strings.Contains(err.Error(), netplay.DefaultPort) {
		t.Errorf("error %q does not mention the default port %s", err, netplay.DefaultPort)
	}
}

func TestJoiningNothingFails(t *testing.T) {
	if _, err := netplay.Join[state, controls]("127.0.0.1:1"); err == nil {
		t.Error("joining a closed port succeeded")
	}
}

func TestClientReportsAHostThatWentAway(t *testing.T) {
	h, err := netplay.Listen[state, controls]("127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	c, err := netplay.Join[state, controls](h.Addr())
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	defer func() { _ = c.Close() }()
	waitFor(t, "the player to connect", h.Joined)

	_ = h.Close()
	waitFor(t, "the client to notice the host has gone", func() bool { return c.Err() != nil })
}

func TestAddrResolvesThePort(t *testing.T) {
	h, err := netplay.Listen[state, controls]("127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = h.Close() }()

	if addr := h.Addr(); strings.HasSuffix(addr, ":0") {
		t.Errorf("Addr = %q, want the port the OS actually assigned", addr)
	}
}

func TestConcurrentSendsAreSafe(t *testing.T) {
	// Run with -race: two goroutines publishing at once must not corrupt the
	// gob stream or trip the detector.
	h, c := hostAndClient(t)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 50 {
			h.Send(state{Tick: i})
		}
	}()
	for i := range 50 {
		h.Send(state{Tick: 100 + i})
	}
	<-done

	waitFor(t, "a snapshot to arrive", func() bool {
		_, ok := c.Snapshot()
		return ok
	})
}
