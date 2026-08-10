package netplay_test

import (
	"strings"
	"testing"

	"github.com/danielriddell21/crucible/netplay"
)

func TestZeroSessionIsOffline(t *testing.T) {
	var s netplay.Session[state, controls]

	if s.Role() != netplay.Offline || s.Online() || s.Peered() {
		t.Errorf("role %v, online %v, peered %v; want a usable offline session",
			s.Role(), s.Online(), s.Peered())
	}
	if got := s.Status(); got != "offline" {
		t.Errorf("Status = %q, want %q", got, "offline")
	}
	if s.Addr() != "" {
		t.Errorf("Addr = %q, want empty when not hosting", s.Addr())
	}
	if err := s.Err(); err != nil {
		t.Errorf("Err = %v, want nil", err)
	}
}

func TestOfflineCallsAreNoOps(t *testing.T) {
	// The game loop should not have to branch on whether a session exists.
	var s netplay.Session[state, controls]

	s.Send(state{Tick: 1})
	s.SendInput(controls{Throttle: 1})
	s.Leave()

	if in, ok := s.Input(); ok || in.Throttle != 0 {
		t.Errorf("Input = %+v, %v; want the zero controls and false", in, ok)
	}
	if st, ok := s.Snapshot(); ok || st.Tick != 0 {
		t.Errorf("Snapshot = %+v, %v; want the zero state and false", st, ok)
	}
	if s.Role() != netplay.Offline {
		t.Errorf("role = %v after no-ops, want offline", s.Role())
	}
}

func TestHostingLobbyLifecycle(t *testing.T) {
	var host netplay.Session[state, controls]
	if err := host.StartHosting("127.0.0.1:0"); err != nil {
		t.Fatalf("StartHosting: %v", err)
	}
	defer host.Leave()

	if host.Role() != netplay.Hosting || !host.Online() {
		t.Fatalf("role = %v, want hosting", host.Role())
	}
	if host.Peered() {
		t.Error("peered with nobody connected")
	}
	if got := host.Status(); !strings.Contains(got, "waiting for a player") {
		t.Errorf("Status = %q, want it to say it is waiting", got)
	}

	var join netplay.Session[state, controls]
	if err := join.StartJoining(host.Addr()); err != nil {
		t.Fatalf("StartJoining: %v", err)
	}
	defer join.Leave()

	waitFor(t, "the host to see the player", host.Peered)
	if got := host.Status(); got != "a player has joined" {
		t.Errorf("Status = %q, want it to report the player", got)
	}
	if join.Role() != netplay.Joining {
		t.Errorf("joiner role = %v, want joining", join.Role())
	}
}

func TestStateFlowsThroughASession(t *testing.T) {
	var host, join netplay.Session[state, controls]
	if err := host.StartHosting("127.0.0.1:0"); err != nil {
		t.Fatalf("StartHosting: %v", err)
	}
	defer host.Leave()
	if err := join.StartJoining(host.Addr()); err != nil {
		t.Fatalf("StartJoining: %v", err)
	}
	defer join.Leave()
	waitFor(t, "the host to see the player", host.Peered)

	join.SendInput(controls{Steer: 0.5})
	waitFor(t, "controls to reach the host", func() bool {
		in, ok := host.Input()
		return ok && in.Steer == 0.5
	})

	host.Send(state{Tick: 42})
	waitFor(t, "state to reach the joiner", func() bool {
		st, ok := join.Snapshot()
		return ok && st.Tick == 42
	})
	if got := join.Status(); got != "connected" {
		t.Errorf("Status = %q, want %q", got, "connected")
	}
}

func TestJoinerNotYetPeeredSaysSo(t *testing.T) {
	var host, join netplay.Session[state, controls]
	if err := host.StartHosting("127.0.0.1:0"); err != nil {
		t.Fatalf("StartHosting: %v", err)
	}
	defer host.Leave()
	if err := join.StartJoining(host.Addr()); err != nil {
		t.Fatalf("StartJoining: %v", err)
	}
	defer join.Leave()

	// Connected at the socket, but the host has sent nothing yet.
	if join.Peered() {
		t.Error("peered before the first snapshot arrived")
	}
	if got := join.Status(); got != "connecting to the host" {
		t.Errorf("Status = %q, want %q", got, "connecting to the host")
	}
}

func TestLeaveReturnsToOffline(t *testing.T) {
	var s netplay.Session[state, controls]
	if err := s.StartHosting("127.0.0.1:0"); err != nil {
		t.Fatalf("StartHosting: %v", err)
	}
	s.Leave()

	if s.Role() != netplay.Offline || s.Online() {
		t.Errorf("role = %v after Leave, want offline", s.Role())
	}
	if s.Addr() != "" {
		t.Errorf("Addr = %q after Leave, want empty", s.Addr())
	}
	// The port is free again, so the same session can host once more.
	if err := s.StartHosting("127.0.0.1:0"); err != nil {
		t.Fatalf("re-hosting after Leave: %v", err)
	}
	s.Leave()
}

func TestStartingTwiceKeepsTheFirstSession(t *testing.T) {
	var s netplay.Session[state, controls]
	if err := s.StartHosting("127.0.0.1:0"); err != nil {
		t.Fatalf("StartHosting: %v", err)
	}
	defer s.Leave()
	first := s.Addr()

	if err := s.StartHosting("127.0.0.1:0"); err != nil {
		t.Errorf("second StartHosting returned %v, want a silent no-op", err)
	}
	if s.Addr() != first {
		t.Errorf("Addr changed from %q to %q; the first session should stand", first, s.Addr())
	}
	if err := s.StartJoining("127.0.0.1:1"); err != nil {
		t.Errorf("StartJoining while hosting returned %v, want a silent no-op", err)
	}
	if s.Role() != netplay.Hosting {
		t.Errorf("role = %v, want it to still be hosting", s.Role())
	}
}

func TestAFailedJoinStaysOfflineAndExplains(t *testing.T) {
	var s netplay.Session[state, controls]
	if err := s.StartJoining("127.0.0.1:1"); err == nil {
		t.Fatal("joining a closed port succeeded")
	}
	if s.Role() != netplay.Offline {
		t.Errorf("role = %v after a failed join, want offline", s.Role())
	}
	if s.Err() == nil {
		t.Error("Err = nil after a failed join")
	}
	if got := s.Status(); !strings.HasPrefix(got, "offline: ") {
		t.Errorf("Status = %q, want it to explain the failure", got)
	}
}

func TestErrSurvivesLeave(t *testing.T) {
	// The lobby is shown after the session ends, so it still needs the reason.
	var s netplay.Session[state, controls]
	_ = s.StartJoining("127.0.0.1:1")
	s.Leave()
	if s.Err() == nil {
		t.Error("Err = nil after Leave, want the reason to survive")
	}
}

func TestTheReasonASessionEndedSurvivesLeave(t *testing.T) {
	// Not only a session that never started: one that connected, lost the other
	// player and was then left has to be able to say what happened, because
	// that is the screen the player is looking at afterwards.
	var host, join netplay.Session[state, controls]
	if err := host.StartHosting("127.0.0.1:0"); err != nil {
		t.Fatalf("StartHosting: %v", err)
	}
	if err := join.StartJoining(host.Addr()); err != nil {
		t.Fatalf("StartJoining: %v", err)
	}
	waitFor(t, "the host to see the player", host.Peered)

	join.Leave()
	waitFor(t, "the host to notice", func() bool { return host.Err() != nil })
	dropped := host.Err()

	host.Leave()
	if got := host.Err(); got == nil {
		t.Errorf("Err = nil after Leave, want %v to survive", dropped)
	}
}

func TestJoinerNoticesTheHostGoingAway(t *testing.T) {
	var host, join netplay.Session[state, controls]
	if err := host.StartHosting("127.0.0.1:0"); err != nil {
		t.Fatalf("StartHosting: %v", err)
	}
	if err := join.StartJoining(host.Addr()); err != nil {
		t.Fatalf("StartJoining: %v", err)
	}
	defer join.Leave()
	waitFor(t, "the host to see the player", host.Peered)

	host.Leave()
	waitFor(t, "the joiner to notice", func() bool { return join.Err() != nil })
	if got := join.Status(); !strings.HasPrefix(got, "lost the host") {
		t.Errorf("Status = %q, want it to say the host went away", got)
	}
}

func TestStatusNamesAReadableAddress(t *testing.T) {
	// A wildcard listen resolves to "[::]:7777", which is no use on screen.
	var s netplay.Session[state, controls]
	if err := s.StartHosting(":0"); err != nil {
		t.Fatalf("StartHosting: %v", err)
	}
	defer s.Leave()

	got := s.Status()
	if strings.Contains(got, "[::]") || strings.Contains(got, "0.0.0.0") {
		t.Errorf("Status = %q, want the wildcard host trimmed away", got)
	}
	if !strings.Contains(got, "port ") {
		t.Errorf("Status = %q, want it to name the port", got)
	}
}

func TestRoleString(t *testing.T) {
	for role, want := range map[netplay.Role]string{
		netplay.Offline: "offline",
		netplay.Hosting: "hosting",
		netplay.Joining: "joining",
	} {
		if got := role.String(); got != want {
			t.Errorf("Role(%d).String() = %q, want %q", role, got, want)
		}
	}
}
