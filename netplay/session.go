package netplay

import (
	"fmt"
	"net"
	"sync"
)

// Role is which end of a session this process is.
type Role int

// The session roles.
const (
	// Offline is a single-player session: nothing is listening and nothing is
	// connected.
	Offline Role = iota
	// Hosting means this process owns the simulation. It may or may not have a
	// player connected yet; [Session.Peered] says which.
	Hosting
	// Joining means this process is driving a character in somebody else's
	// simulation.
	Joining
)

// String returns the role's name.
func (r Role) String() string {
	switch r {
	case Hosting:
		return "hosting"
	case Joining:
		return "joining"
	default:
		return "offline"
	}
}

// Session is the lobby-facing wrapper around a [Host] and a [Client]: one
// object a menu can bind to, which is in exactly one of the three [Role]
// states and reports a line of text describing it.
//
// It exists because the awkward part of adding multiplayer to a game is rarely
// the socket — it is that the title screen, the pause menu and the game loop
// all need to agree on whether there is a session, who owns it, and what to
// tell the player when it fails. Session holds that, so a lobby screen is a
// menu bound to [Session.StartHosting], [Session.StartJoining] and
// [Session.Leave], drawing [Session.Status].
//
// The zero value is a usable offline session. Every method is safe to call
// from the game loop while the connection works in the background, and calling
// one in the wrong state is a no-op rather than an error: leaving an offline
// session does nothing, and hosting twice keeps the first.
type Session[S, I any] struct {
	mu     sync.Mutex
	role   Role
	host   *Host[S, I]
	client *Client[S, I]
	failed error
}

// StartHosting begins listening on addr, which may name only a port
// (":7777"). An empty addr listens on [DefaultPort] on every interface.
// A session that is already hosting or joining is left alone.
func (s *Session[S, I]) StartHosting(addr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.role != Offline {
		return nil
	}
	if addr == "" {
		addr = ":" + DefaultPort
	}
	h, err := Listen[S, I](addr)
	if err != nil {
		s.failed = err
		return err
	}
	s.host, s.role, s.failed = h, Hosting, nil
	return nil
}

// StartJoining connects to a host at addr, which may omit the port. A session
// that is already hosting or joining is left alone.
func (s *Session[S, I]) StartJoining(addr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.role != Offline {
		return nil
	}
	c, err := Join[S, I](addr)
	if err != nil {
		s.failed = err
		return err
	}
	s.client, s.role, s.failed = c, Joining, nil
	return nil
}

// Leave ends the session and returns to [Offline], keeping the last error so
// the lobby can still say why a session ended. It is a no-op when offline.
func (s *Session[S, I]) Leave() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.host != nil {
		_ = s.host.Close()
		s.host = nil
	}
	if s.client != nil {
		_ = s.client.Close()
		s.client = nil
	}
	s.role = Offline
}

// Role returns which end of the session this process is.
func (s *Session[S, I]) Role() Role {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.role
}

// Online reports whether a session is running at all, whichever end of it this
// process is.
func (s *Session[S, I]) Online() bool { return s.Role() != Offline }

// Peered reports whether the other player is actually connected. A host that
// is listening with nobody there is [Hosting] but not peered, which is the
// state a lobby spends most of its time in.
func (s *Session[S, I]) Peered() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case s.host != nil:
		return s.host.Joined()
	case s.client != nil:
		_, got := s.client.Snapshot()
		return got && s.client.Err() == nil
	default:
		return false
	}
}

// Addr returns the address the session is listening on when hosting, and an
// empty string otherwise. Show it to the player: it is what the other machine
// needs to type in.
func (s *Session[S, I]) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.host == nil {
		return ""
	}
	return s.host.Addr()
}

// Send publishes state to the joining player. It is meaningful only while
// hosting, and a no-op otherwise, so a game loop that always calls it needs no
// branch of its own.
func (s *Session[S, I]) Send(state S) {
	s.mu.Lock()
	h := s.host
	s.mu.Unlock()
	if h != nil {
		h.Send(state)
	}
}

// SendInput publishes this player's controls to the host. It is meaningful
// only while joining, and a no-op otherwise.
func (s *Session[S, I]) SendInput(in I) {
	s.mu.Lock()
	c := s.client
	s.mu.Unlock()
	if c != nil {
		c.Send(in)
	}
}

// Input returns the joining player's controls, for a host to apply to the
// character they are driving. The second result is false when there is nobody
// to take input from, which is the cue to leave that character to the
// simulation.
func (s *Session[S, I]) Input() (I, bool) {
	s.mu.Lock()
	h := s.host
	s.mu.Unlock()
	if h == nil {
		var zero I
		return zero, false
	}
	return h.Input()
}

// Snapshot returns the most recent state the host sent, for a joining player
// to draw. The second result is false until the first one arrives.
func (s *Session[S, I]) Snapshot() (S, bool) {
	s.mu.Lock()
	c := s.client
	s.mu.Unlock()
	if c == nil {
		var zero S
		return zero, false
	}
	return c.Snapshot()
}

// Err returns the error that ended or prevented the session, if any. A host
// reports a player dropping out; a client reports the host going away. It
// survives [Session.Leave] so a lobby can explain what happened.
func (s *Session[S, I]) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case s.host != nil:
		if err := s.host.Err(); err != nil {
			return err
		}
	case s.client != nil:
		if err := s.client.Err(); err != nil {
			return err
		}
	}
	return s.failed
}

// Status returns one line describing the session, for a lobby screen to draw.
// The wording is deliberately plain and player-facing: an address to read out,
// a state to wait through, or a reason it did not work.
func (s *Session[S, I]) Status() string {
	role, peered, addr, err := s.Role(), s.Peered(), s.Addr(), s.Err()
	switch role {
	case Hosting:
		if peered {
			return "a player has joined"
		}
		if err != nil {
			return fmt.Sprintf("waiting on %s (last player left: %v)", displayAddr(addr), err)
		}
		return fmt.Sprintf("waiting for a player on %s", displayAddr(addr))
	case Joining:
		if peered {
			return "connected"
		}
		if err != nil {
			return fmt.Sprintf("lost the host: %v", err)
		}
		return "connecting to the host"
	default:
		if err != nil {
			return fmt.Sprintf("offline: %v", err)
		}
		return "offline"
	}
}

// displayAddr trims a wildcard listen address down to what a player needs to
// read out. Go reports ":7777" as "[::]:7777", which is not something to put
// on a lobby screen.
func displayAddr(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if host == "" || host == "::" || host == "0.0.0.0" {
		return "port " + port
	}
	return addr
}
