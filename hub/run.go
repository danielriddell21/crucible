package hub

import (
	"encoding/json"
	"fmt"
	"os"
)

// RunLeader starts a hub for the leader window: run receives the leader's
// Link and blocks until the window closes (conventionally by calling the
// application's gui.Run). Child windows spawned along the way are killed on
// return.
func RunLeader[M any](cfg Config[M], run func(Link[M]) error) error {
	h := New(cfg)
	leaderIn := make(chan M, 64)  // hub -> leader (the leader's participant out)
	leaderOut := make(chan M, 64) // leader -> hub
	h.AddParticipant(leaderIn, nil)
	go h.Run()
	go func() {
		for m := range leaderOut {
			h.Inject(0, m)
		}
	}()
	err := run(Link[M]{In: leaderIn, Out: leaderOut})
	close(leaderOut)
	h.Shutdown()
	if err != nil {
		return fmt.Errorf("run leader window: %w", err)
	}
	return nil
}

// RunChild runs a child window wired to the leader over stdin/stdout: run
// receives the child's Link and blocks until the window closes. The Link's
// In channel closes when the leader disappears.
func RunChild[M any](run func(Link[M]) error) error {
	in := make(chan M, 64)
	out := make(chan M, 64)
	go func() { // leader stdin -> in
		dec := json.NewDecoder(os.Stdin)
		for {
			var m M
			if dec.Decode(&m) != nil {
				break
			}
			in <- m
		}
		close(in) // leader gone: closing In makes the window terminate
	}()
	go func() { // out -> leader via stdout
		enc := json.NewEncoder(os.Stdout)
		for m := range out {
			if enc.Encode(m) != nil {
				break
			}
		}
	}()
	if err := run(Link[M]{In: in, Out: out}); err != nil {
		return fmt.Errorf("run child window: %w", err)
	}
	return nil
}
