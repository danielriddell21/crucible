package level_test

import (
	"path/filepath"
	"testing"

	"github.com/danielriddell21/crucible/geom"
	"github.com/danielriddell21/crucible/level"
	"github.com/danielriddell21/crucible/raycast"
	"github.com/danielriddell21/crucible/record"
	"github.com/danielriddell21/crucible/worldgen"
)

// framePainter is a minimal raycast.ColumnPainter: it shades every span by
// distance into an RGBA framebuffer, so the integration test can prove the
// world and renderer compose without any texture assets. It also tallies
// which span kinds it was asked to draw.
type framePainter struct {
	fb                   []byte
	w, h                 int
	walls, floors, ceils int
}

func newFramePainter(w, h int) *framePainter {
	return &framePainter{fb: make([]byte, w*h*4), w: w, h: h}
}

func (p *framePainter) shade(dist float64) uint8 {
	v := 255.0 / (1 + dist*0.15)
	if v > 255 {
		v = 255
	}
	return uint8(v)
}

func (p *framePainter) put(x, y int, r, g, b uint8) {
	if x < 0 || y < 0 || x >= p.w || y >= p.h {
		return
	}
	i := (y*p.w + x) * 4
	p.fb[i], p.fb[i+1], p.fb[i+2], p.fb[i+3] = r, g, b, 255
}

func (p *framePainter) WallSpan(x, y0, y1 int, f raycast.Face) {
	p.walls++
	g := p.shade(f.Dist)
	if f.Side == 1 {
		g = uint8(int(g) * 8 / 10) // darken north/south faces
	}
	for y := y0; y <= y1; y++ {
		p.put(x, y, g, g, g)
	}
}

func (p *framePainter) FloorSpan(x, y0, y1 int, _ geom.Coord, _, dist float64) {
	p.floors++
	c := p.shade(dist)
	for y := y0; y <= y1; y++ {
		p.put(x, y, c, uint8(int(c)*6/10), uint8(int(c)*4/10))
	}
}

func (p *framePainter) CeilSpan(x, y0, y1 int, _ geom.Coord, _, dist float64) {
	p.ceils++
	c := p.shade(dist) / 2
	for y := y0; y <= y1; y++ {
		p.put(x, y, c/2, c/2, c)
	}
}

const (
	frameW, frameH = 320, 200
	eyeHeight      = 0.5
)

// renderFrame runs the whole pipeline: it generates a furnished level, puts
// the camera at the spawn, and walks every screen column, returning the
// painter and the per-column depth buffer.
func renderFrame(t *testing.T, seed int64) (*level.Level, *framePainter, []float64) {
	t.Helper()
	passes := []level.Pass{
		func(l *level.Level, rng *worldgen.RNG, rooms []geom.Rect) {
			level.AssignHeights(l, rng, rooms, level.HeightsConfig{})
			level.PlaceLowWalls(l, rng, level.LowWallConfig{})
			level.PlaceDoors(l, rng, level.DoorConfig{})
			level.CarveVents(l, rooms, level.VentConfig{})
			level.AssignThemes(l, rng, rooms, 3)
		},
	}
	l, _, err := level.Generate(level.GenerateConfig{Width: 48, Height: 40, Seed: seed}, passes, nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	cam := raycast.NewCamera(l.Spawn.Vec2(), 0, 1.05)
	p := newFramePainter(frameW, frameH)
	zbuf := make([]float64, frameW)
	for x := range frameW {
		res := raycast.WalkColumn(cam, frameW, frameH, x, eyeHeight, l, p)
		zbuf[x] = res.Depth
	}
	return l, p, zbuf
}

func TestRenderPipelineComposes(t *testing.T) {
	_, p, zbuf := renderFrame(t, 1)

	// Every column is enclosed by the level's border, so every ray closes at
	// a finite, positive depth.
	for x, d := range zbuf {
		if d <= 0 || d > 1e6 {
			t.Fatalf("column %d closed at implausible depth %v", x, d)
		}
	}

	// The painter drew all three span kinds — floors, ceilings, and walls.
	if p.walls == 0 || p.floors == 0 || p.ceils == 0 {
		t.Fatalf("spans: %d walls %d floors %d ceils", p.walls, p.floors, p.ceils)
	}

	// The framebuffer is fully painted (no transparent gaps) and not a flat
	// fill (walls, floor, and ceiling produce distinct colours).
	opaque, distinct := 0, map[[3]byte]bool{}
	for i := 0; i+3 < len(p.fb); i += 4 {
		if p.fb[i+3] == 255 {
			opaque++
		}
		distinct[[3]byte{p.fb[i], p.fb[i+1], p.fb[i+2]}] = true
	}
	if opaque != frameW*frameH {
		t.Fatalf("frame has %d/%d opaque pixels — gaps in the column fill", opaque, frameW*frameH)
	}
	if len(distinct) < 10 {
		t.Fatalf("frame has only %d distinct colours — render looks flat", len(distinct))
	}
}

func TestRenderPipelineDeterministic(t *testing.T) {
	_, a, _ := renderFrame(t, 7)
	_, b, _ := renderFrame(t, 7)
	for i := range a.fb {
		if a.fb[i] != b.fb[i] {
			t.Fatal("same seed produced a different frame")
		}
	}
}

func TestRenderFrameEncodesToPNG(t *testing.T) {
	_, p, _ := renderFrame(t, 3)
	path := filepath.Join(t.TempDir(), "frame.png")
	if err := record.SavePNG(path, record.FromRGBA(p.fb, frameW, frameH)); err != nil {
		t.Fatalf("SavePNG: %v", err)
	}
}
