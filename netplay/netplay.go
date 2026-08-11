// Package netplay carries a two-player session over the network: one machine
// hosts and owns the simulation, the other joins, sends its controls, and
// draws what the host tells it.
//
// The model is deliberately the simple one. The host's word is final, there is
// no prediction and no rollback, and the whole moving state goes over the wire
// as one snapshot rather than as deltas. That suits a game where a little
// latency on the other player is cosmetic — a chase, a co-op wander, a shared
// sandbox — and it is far less machinery than a fighting game or a shooter
// would need. A game that cannot tolerate the lag wants a different package,
// not more options on this one.
//
// Both the snapshot and the input type belong to the game; a [Host] and a
// [Client] are generic over them and encode them with [encoding/gob], so the
// engine never grows a vocabulary of poses or buttons. Keep out of the
// snapshot anything both ends can derive: a world generated from a seed is
// built identically on each machine, and sending it every tick would be the
// bulk of the traffic for no gain.
//
// Type parameters cannot be inferred from a constructor's arguments, so name
// them at the call:
//
//	host, err := netplay.Listen[Snapshot, Input](":7777")
//	peer, err := netplay.Join[Snapshot, Input]("192.168.1.20")
//
// Both sides are safe to use from the game loop: the read side runs on its own
// goroutine and every accessor is guarded.
package netplay

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

// DefaultPort is the port [Join] assumes when an address names only a host.
const DefaultPort = "7777"

// DialTimeout bounds how long [Join] waits for a host to answer.
const DialTimeout = 8 * time.Second

// Host owns the simulation and accepts a single joining player. S is the
// snapshot type it publishes, I the input type it receives.
type Host[S, I any] struct {
	listener net.Listener

	// send serialises writes to the encoder. It is separate from mu so a slow
	// or blocked socket cannot hold up a caller reading Input from the game
	// loop.
	send sync.Mutex

	mu      sync.Mutex
	conn    net.Conn
	enc     *gob.Encoder
	input   I
	joined  bool
	lastErr error
}

// Listen starts a host on addr, which may name only a port (":7777"). The
// returned host accepts connections in the background; call [Host.Joined] to
// see whether anyone has arrived.
func Listen[S, I any](addr string) (*Host[S, I], error) {
	var cfg net.ListenConfig
	l, err := cfg.Listen(context.Background(), "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("netplay: listen on %s: %w", addr, err)
	}
	h := &Host[S, I]{listener: l}
	go h.accept()
	return h, nil
}

// Addr returns the address the host is listening on, with the port resolved.
func (h *Host[S, I]) Addr() string { return h.listener.Addr().String() }

// Joined reports whether a player is currently connected.
func (h *Host[S, I]) Joined() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.joined
}

// Input returns the joining player's most recent controls, and whether anyone
// is connected to have sent them. A player who has not arrived, or who has
// gone quiet, simply sends nothing — what the host does about that (leave the
// character standing, hand it back to the simulation) is the game's decision.
func (h *Host[S, I]) Input() (I, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.input, h.joined
}

// Send publishes the current state to the joining player. It is a no-op when
// nobody is connected, so the game loop can call it unconditionally.
func (h *Host[S, I]) Send(s S) {
	h.send.Lock()
	defer h.send.Unlock()

	h.mu.Lock()
	enc, conn := h.enc, h.conn
	h.mu.Unlock()
	if enc == nil {
		return
	}
	if err := enc.Encode(s); err != nil {
		h.drop(conn, err)
	}
}

// Err returns the last connection error, if one has occurred. A player
// disconnecting is reported here; the host keeps listening for the next.
func (h *Host[S, I]) Err() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.lastErr
}

// Close stops listening and disconnects any player.
func (h *Host[S, I]) Close() error {
	h.mu.Lock()
	conn := h.conn
	// Nobody is joined to a host that has shut down, and saying so here rather
	// than waiting for the reader goroutine to notice keeps [Host.Joined]
	// honest the instant Close returns.
	h.conn, h.enc, h.joined = nil, nil, false
	h.mu.Unlock()
	if conn != nil {
		_ = conn.Close()
	}
	if err := h.listener.Close(); err != nil {
		return fmt.Errorf("netplay: close listener: %w", err)
	}
	return nil
}

func (h *Host[S, I]) accept() {
	for {
		conn, err := h.listener.Accept()
		if err != nil {
			return // the listener was closed
		}
		h.mu.Lock()
		// One player at a time: a second caller replaces the first, which is
		// also how a player who dropped gets back in.
		if h.conn != nil {
			_ = h.conn.Close()
		}
		h.conn, h.enc, h.joined = conn, gob.NewEncoder(conn), true
		h.mu.Unlock()
		go h.read(conn)
	}
}

func (h *Host[S, I]) read(conn net.Conn) {
	dec := gob.NewDecoder(conn)
	for {
		var in I
		if err := dec.Decode(&in); err != nil {
			h.drop(conn, err)
			return
		}
		h.mu.Lock()
		h.input = in
		h.mu.Unlock()
	}
}

// drop disconnects conn and forgets the player driving it. It does nothing to
// the host's state when conn is no longer the live connection: accepting a
// replacement closes the old socket, and the reader that was blocked on it
// wakes with an error a moment later. Clearing on that late error would wipe
// out the player who has just arrived.
func (h *Host[S, I]) drop(conn net.Conn, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !errors.Is(err, net.ErrClosed) {
		h.lastErr = err
	}
	if conn == nil || h.conn != conn {
		return
	}
	_ = conn.Close()
	var zero I
	h.conn, h.enc, h.joined, h.input = nil, nil, false, zero
}

// Client is the joining side: it sends controls and receives snapshots. S and
// I match the host's.
type Client[S, I any] struct {
	conn net.Conn

	send sync.Mutex
	enc  *gob.Encoder

	mu     sync.Mutex
	latest S
	got    bool
	err    error
}

// Join connects to a host. An address naming only a host gets [DefaultPort].
func Join[S, I any](addr string) (*Client[S, I], error) {
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = net.JoinHostPort(addr, DefaultPort)
	}
	d := net.Dialer{Timeout: DialTimeout}
	conn, err := d.DialContext(context.Background(), "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("netplay: join %s: %w", addr, err)
	}
	c := &Client[S, I]{conn: conn, enc: gob.NewEncoder(conn)}
	go c.read()
	return c, nil
}

// Send publishes this player's controls to the host.
func (c *Client[S, I]) Send(in I) {
	c.send.Lock()
	defer c.send.Unlock()
	if err := c.enc.Encode(in); err != nil {
		c.fail(err)
	}
}

// Snapshot returns the most recent state the host sent, and whether one has
// arrived yet. Before the first snapshot the zero value is returned, which is
// the cue to draw a "connecting" screen rather than a world.
func (c *Client[S, I]) Snapshot() (S, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.latest, c.got
}

// Err returns the error that ended the connection, if it has ended.
func (c *Client[S, I]) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

// Close disconnects from the host.
func (c *Client[S, I]) Close() error {
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("netplay: close connection: %w", err)
	}
	return nil
}

func (c *Client[S, I]) read() {
	dec := gob.NewDecoder(c.conn)
	for {
		var s S
		if err := dec.Decode(&s); err != nil {
			c.fail(err)
			return
		}
		c.mu.Lock()
		c.latest, c.got = s, true
		c.mu.Unlock()
	}
}

func (c *Client[S, I]) fail(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err == nil && !errors.Is(err, net.ErrClosed) {
		c.err = err
	}
}
