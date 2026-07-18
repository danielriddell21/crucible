package store_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielriddell21/crucible/store"
)

type doc struct {
	Name  string  `json:"name"`
	Score float64 `json:"score"`
}

func TestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app", "doc.json")
	want := doc{Name: "run", Score: 12.5}
	if err := store.Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got := store.Load(path, doc{})
	if got != want {
		t.Fatalf("Load = %+v, want %+v", got, want)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !strings.HasSuffix(string(data), "\n") {
		t.Error("saved document must end with a newline")
	}
}

func TestLoadMissingReturnsDefault(t *testing.T) {
	def := doc{Name: "default"}
	if got := store.Load(filepath.Join(t.TempDir(), "absent.json"), def); got != def {
		t.Fatalf("Load = %+v", got)
	}
}

func TestLoadCorruptReturnsDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	def := doc{Name: "default", Score: 1}
	if got := store.Load(path, def); got != def {
		t.Fatalf("Load = %+v", got)
	}
}

func TestLoadPartialKeepsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "partial.json")
	if err := os.WriteFile(path, []byte(`{"name":"only"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got := store.Load(path, doc{Name: "x", Score: 3})
	if got.Name != "only" || got.Score != 3 {
		t.Fatalf("Load = %+v", got)
	}
}

func TestEmptyPath(t *testing.T) {
	def := doc{Name: "default"}
	if got := store.Load("", def); got != def {
		t.Fatalf("Load(\"\") = %+v", got)
	}
	if err := store.Save("", def); err != nil {
		t.Fatalf("Save(\"\") = %v", err)
	}
}

func TestPath(t *testing.T) {
	p, err := store.Path("crucible", "settings.json")
	if err != nil {
		t.Skipf("no user config dir here: %v", err)
	}
	if filepath.Base(p) != "settings.json" || filepath.Base(filepath.Dir(p)) != "crucible" {
		t.Fatalf("Path = %q", p)
	}
}
