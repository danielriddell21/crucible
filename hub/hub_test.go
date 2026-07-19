package hub_test

import (
	"os/exec"
	"testing"
	"time"

	"github.com/danielriddell21/crucible/hub"
)

type msg struct {
	Type string `json:"t"`
	N    int    `json:"n,omitempty"`
}

func route(m msg) hub.Route {
	switch m.Type {
	case "state":
		return hub.RouteState
	case "poke":
		return hub.RouteBroadcast
	case "add":
		return hub.RouteSpawn
	case "remove":
		return hub.RouteCloseNewest
	}
	return hub.RouteNone
}

func newHub(t *testing.T) *hub.Hub[msg] {
	t.Helper()
	h := hub.New(hub.Config[msg]{
		Self:      "unused",
		ChildArgs: func(idx int) []string { return nil },
		Route:     route,
		Quit:      msg{Type: "quit"},
	})
	go h.Run()
	t.Cleanup(h.Shutdown)
	return h
}

func recv(t *testing.T, ch chan msg) msg {
	t.Helper()
	select {
	case m := <-ch:
		return m
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
		return msg{}
	}
}

func expectNone(t *testing.T, ch chan msg) {
	t.Helper()
	select {
	case m := <-ch:
		t.Fatalf("unexpected message %+v", m)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestBroadcastSkipsSource(t *testing.T) {
	h := newHub(t)
	a := make(chan msg, 8)
	b := make(chan msg, 8)
	idA := h.AddParticipant(a, nil)
	h.AddParticipant(b, nil)

	h.Inject(idA, msg{Type: "poke", N: 1})
	if got := recv(t, b); got.N != 1 {
		t.Fatalf("b got %+v", got)
	}
	expectNone(t, a)
}

func TestStateReplayedToLateJoiner(t *testing.T) {
	h := newHub(t)
	a := make(chan msg, 8)
	idA := h.AddParticipant(a, nil)
	h.Inject(idA, msg{Type: "state", N: 42})

	// Give the hub a moment to cache the state before the join.
	time.Sleep(20 * time.Millisecond)
	late := make(chan msg, 8)
	h.AddParticipant(late, nil)
	if got := recv(t, late); got.Type != "state" || got.N != 42 {
		t.Fatalf("late joiner got %+v", got)
	}
}

func TestRouteNoneIgnored(t *testing.T) {
	h := newHub(t)
	a := make(chan msg, 8)
	b := make(chan msg, 8)
	idA := h.AddParticipant(a, nil)
	h.AddParticipant(b, nil)
	h.Inject(idA, msg{Type: "mystery"})
	expectNone(t, b)
}

func TestCloseNewestQuitsLatestChild(t *testing.T) {
	h := newHub(t)
	leader := make(chan msg, 8)
	h.AddParticipant(leader, nil)
	older := make(chan msg, 8)
	newer := make(chan msg, 8)
	h.AddParticipant(older, &exec.Cmd{})
	h.AddParticipant(newer, &exec.Cmd{})

	h.CloseNewest()
	if got := recv(t, newer); got.Type != "quit" {
		t.Fatalf("newest got %+v", got)
	}
	expectNone(t, older)
	expectNone(t, leader)
}

func TestCloseNewestWithoutChildren(t *testing.T) {
	h := newHub(t)
	leader := make(chan msg, 8)
	h.AddParticipant(leader, nil)
	h.CloseNewest() // must not panic or message the leader
	expectNone(t, leader)
}

func TestRunLeaderLifecycle(t *testing.T) {
	cfg := hub.Config[msg]{
		Self:      "unused",
		ChildArgs: func(int) []string { return nil },
		Route:     route,
		Quit:      msg{Type: "quit"},
	}
	ran := false
	err := hub.RunLeader(cfg, func(l hub.Link[msg]) error {
		ran = true
		l.Out <- msg{Type: "state", N: 7} // exercises the pump goroutine
		return nil
	})
	if err != nil || !ran {
		t.Fatalf("RunLeader: err=%v ran=%v", err, ran)
	}
}

func TestTrySendDropsWhenFull(t *testing.T) {
	ch := make(chan msg, 1)
	hub.TrySend(ch, msg{N: 1})
	hub.TrySend(ch, msg{N: 2}) // must not block
	if got := <-ch; got.N != 1 {
		t.Fatalf("got %+v", got)
	}
}
