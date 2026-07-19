package hub_test

import (
	"fmt"
	"os"

	"github.com/danielriddell21/crucible/hub"
)

// Msg is the application's own wire type; the hub is generic over it.
type Msg struct {
	Type string  `json:"t"`
	Yaw  float32 `json:"yaw,omitempty"`
}

// ExampleRunLeader shows the leader side of a multi-window visualizer: the
// Route policy maps the app's message types onto hub behaviours, and the
// window loop talks through the Link. Child processes run the same binary
// with ExampleRunChild's wiring.
func ExampleRunLeader() {
	cfg := hub.Config[Msg]{
		Self: os.Args[0],
		ChildArgs: func(idx int) []string {
			return []string{"view", fmt.Sprintf("--child=%d", idx)}
		},
		Route: func(m Msg) hub.Route {
			switch m.Type {
			case "state":
				return hub.RouteState
			case "add":
				return hub.RouteSpawn
			case "remove":
				return hub.RouteCloseNewest
			}
			return hub.RouteNone
		},
		Quit: Msg{Type: "quit"},
	}

	err := hub.RunLeader(cfg, func(l hub.Link[Msg]) error {
		// gui.Run(gui.Config{Link: l, ...}) — the window sends its state on
		// l.Out and applies messages arriving on l.In.
		return nil
	})
	if err != nil {
		fmt.Println(err)
	}
}

// ExampleRunChild is the counterpart run by spawned windows: the Link
// speaks to the leader over stdin/stdout, and In closes when the leader
// goes away.
func ExampleRunChild() {
	err := hub.RunChild(func(l hub.Link[Msg]) error {
		for m := range l.In {
			_ = m // apply the shared state to this window
		}
		return nil
	})
	if err != nil {
		fmt.Println(err)
	}
}
