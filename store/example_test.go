package store_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/danielriddell21/crucible/store"
)

// Example round-trips a settings document the way the family's games do on
// startup and shutdown. In a real game the path comes from [store.Path].
func Example() {
	type settings struct {
		Sound bool    `json:"sound"`
		FOV   float64 `json:"fov"`
	}
	defaults := settings{Sound: true, FOV: 1.152}

	dir, _ := os.MkdirTemp("", "crucible-store-example")
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "demo", "settings.json")

	first := store.Load(path, defaults) // no file yet: defaults
	first.FOV = 1.3
	if err := store.Save(path, first); err != nil {
		fmt.Println("save:", err)
		return
	}

	second := store.Load(path, defaults)
	fmt.Println(second.Sound, second.FOV)
	// Output: true 1.3
}
