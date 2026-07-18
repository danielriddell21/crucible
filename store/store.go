// Package store persists small per-application JSON documents — settings,
// records, saved state — under the user's configuration directory, following
// the family convention of one <app>/<file>.json per document.
//
// [Path] resolves the conventional location, [Load] reads a document over
// the caller's defaults, and [Save] writes one back. Loading is forgiving:
// a missing, unreadable, or corrupt file yields the defaults, so a game
// always starts. Saving is strict and reports errors.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Path returns the conventional location of a document for the named
// application, e.g. Path("nemesis", "records.json").
func Path(app, file string) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config dir: %w", err)
	}
	return filepath.Join(dir, app, file), nil
}

// Load reads the JSON document at path into a copy of def and returns it.
// An empty path, a missing file, or a corrupt document returns def
// unchanged.
func Load[T any](path string, def T) T {
	if path == "" {
		return def
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return def
	}
	v := def
	if err := json.Unmarshal(data, &v); err != nil {
		return def
	}
	return v
}

// Save writes v as indented JSON to path, creating the parent directory if
// needed. An empty path is a no-op, so callers that failed to resolve a path
// at startup can still call Save unconditionally.
func Save[T any](path string, v T) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", filepath.Base(path), err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	return nil
}
