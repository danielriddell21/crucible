// Package hub coordinates a family of visualizer windows: one leader
// process spawns child windows of its own executable and relays JSON
// messages between them over the children's stdin and stdout, so every
// window shows the same state.
//
// The message type belongs to each application; a [Hub] is generic over
// it. A [Config] supplies a routing policy that sorts each inbound message
// into a [Route]: [RouteState] caches and rebroadcasts shared state,
// [RouteBroadcast] fans a message out once, and [RouteSpawn] and
// [RouteCloseNewest] open and close windows. Most applications never touch
// a [Hub] directly: [RunLeader] wires up the leader window and [RunChild]
// a spawned one, each handing the window a [Link] to speak through.
// Delivery is lossy by design through [TrySend], so a stalled window never
// stalls the hub.
package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/danielriddell21/ordinex/v2"
)

// Route says what the hub should do with an inbound message.
type Route int

const (
	// RouteNone ignores the message.
	RouteNone Route = iota
	// RouteState caches the message as the latest shared state, replays it
	// to windows that join later, and broadcasts it to every other window.
	RouteState
	// RouteBroadcast forwards the message to every other window once.
	RouteBroadcast
	// RouteSpawn opens another child window.
	RouteSpawn
	// RouteCloseNewest asks the most recently spawned child window to quit.
	RouteCloseNewest
)

// Config describes how a Hub runs. M is the application's message type,
// which must marshal to JSON.
type Config[M any] struct {
	// Self is the executable to spawn for child windows, conventionally
	// os.Args[0].
	Self string
	// ChildArgs returns the arguments for the idx-th child window, e.g.
	// {"view", "--child=1"}.
	ChildArgs func(idx int) []string
	// Route classifies inbound messages.
	Route func(M) Route
	// Quit is the message sent to a child window to make it close.
	Quit M
	// MaxWindows caps the number of simultaneous windows. Zero means the
	// conventional 16.
	MaxWindows int
}

// DefaultMaxWindows is the window cap applied when Config.MaxWindows is
// zero.
const DefaultMaxWindows = 16

// Hub relays messages between the leader window and its children. Create
// one with [New], register the leader with [Hub.AddParticipant], then start
// [Hub.Run] in a goroutine.
type Hub[M any] struct {
	cfg     Config[M]
	inbox   chan srcMsg[M]
	eof     chan int
	done    chan struct{}
	mu      sync.Mutex
	parts   map[int]*participant[M]
	nextID  int
	spawned int
	last    M
	haveSt  bool
}

type participant[M any] struct {
	out chan M
	cmd *exec.Cmd
}

type srcMsg[M any] struct {
	id int
	m  M
}

// New returns a hub with no participants.
func New[M any](cfg Config[M]) *Hub[M] {
	if cfg.MaxWindows <= 0 {
		cfg.MaxWindows = DefaultMaxWindows
	}
	return &Hub[M]{
		cfg:   cfg,
		inbox: make(chan srcMsg[M], 128),
		eof:   make(chan int, DefaultMaxWindows),
		done:  make(chan struct{}),
		parts: map[int]*participant[M]{},
	}
}

// AddParticipant registers a window that receives messages on out and
// returns its id. cmd is the child process behind the window, or nil for
// the in-process leader. A late joiner immediately receives the cached
// state, if any.
func (h *Hub[M]) AddParticipant(out chan M, cmd *exec.Cmd) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	id := h.nextID
	h.nextID++
	h.parts[id] = &participant[M]{out: out, cmd: cmd}
	if h.haveSt {
		TrySend(out, h.last)
	}
	return id
}

// Inject feeds a message from participant src into the hub, as if it had
// arrived from that window's process.
func (h *Hub[M]) Inject(src int, m M) {
	h.inbox <- srcMsg[M]{src, m}
}

// Run dispatches messages until Shutdown. Run it in its own goroutine.
func (h *Hub[M]) Run() {
	// Stop on shutdown rather than ranging over inbox: child reader
	// goroutines may still send after teardown, so inbox is never closed.
	for {
		select {
		case sm := <-h.inbox:
			h.handle(sm.id, sm.m)
		case id := <-h.eof:
			h.drop(id)
		case <-h.done:
			return
		}
	}
}

func (h *Hub[M]) handle(src int, m M) {
	switch h.cfg.Route(m) {
	case RouteState:
		h.mu.Lock()
		h.last, h.haveSt = m, true
		h.mu.Unlock()
		h.broadcastExcept(src, m)
	case RouteBroadcast:
		h.broadcastExcept(src, m)
	case RouteSpawn:
		h.SpawnChild()
	case RouteCloseNewest:
		h.CloseNewest()
	case RouteNone:
	}
}

func (h *Hub[M]) broadcastExcept(src int, m M) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, p := range h.parts {
		if id != src {
			TrySend(p.out, m)
		}
	}
}

func (h *Hub[M]) drop(id int) {
	h.mu.Lock()
	p := h.parts[id]
	delete(h.parts, id)
	if p != nil {
		close(p.out)
	}
	h.mu.Unlock()
	if p != nil && p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
}

// CloseNewest sends the quit message to the most recently spawned child
// window, if any.
func (h *Hub[M]) CloseNewest() {
	h.mu.Lock()
	ids := make([]int, 0, len(h.parts))
	for id, p := range h.parts {
		if p.cmd != nil {
			ids = append(ids, id)
		}
	}
	h.mu.Unlock()
	if len(ids) == 0 {
		return
	}
	newest := ordinex.MergeSorter[int]{}.Sort(ids)[len(ids)-1]
	h.mu.Lock()
	p := h.parts[newest]
	h.mu.Unlock()
	if p != nil {
		TrySend(p.out, h.cfg.Quit)
	}
}

// SpawnChild starts another child window process and wires it into the hub,
// unless the window cap is reached.
func (h *Hub[M]) SpawnChild() {
	h.mu.Lock()
	if len(h.parts) >= h.cfg.MaxWindows {
		h.mu.Unlock()
		return
	}
	h.spawned++
	idx := h.spawned
	h.mu.Unlock()

	cmd := exec.CommandContext(context.Background(), h.cfg.Self, h.cfg.ChildArgs(idx)...)
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "spawn child window: stdin pipe: %v\n", err)
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "spawn child window: stdout pipe: %v\n", err)
		return
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "spawn child window: start: %v\n", err)
		return
	}

	out := make(chan M, 64)
	id := h.AddParticipant(out, cmd)

	go func() { // hub -> child stdin
		enc := json.NewEncoder(stdin)
		for m := range out {
			if enc.Encode(m) != nil {
				break
			}
		}
		_ = stdin.Close()
	}()
	go func() { // child stdout -> hub, then signal removal and reap
		dec := json.NewDecoder(stdout)
		for {
			var m M
			if dec.Decode(&m) != nil {
				break
			}
			h.Inject(id, m)
		}
		h.eof <- id
		_ = cmd.Wait()
	}()
}

// Shutdown kills every child window process and stops Run.
func (h *Hub[M]) Shutdown() {
	h.mu.Lock()
	cmds := make([]*exec.Cmd, 0, len(h.parts))
	for _, p := range h.parts {
		if p.cmd != nil {
			cmds = append(cmds, p.cmd)
		}
	}
	h.mu.Unlock()
	for _, c := range cmds {
		if c.Process != nil {
			_ = c.Process.Kill()
		}
	}
	close(h.done)
}
