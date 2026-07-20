// Package level is the shared world model behind the family's first-person
// games: a tile grid with per-cell floor and ceiling heights, half-height
// walls, light, sky, and lifts, plus the placement passes that furnish a
// generated dungeon — heights and ledges, low walls, doors, and vent
// networks.
//
// [Generate] runs the whole pipeline: it digs rooms and corridors into a
// fresh [Level], places the spawn and the farthest exit with
// [PlaceSpawnExit], runs the furnishing passes it is given, and retries
// with derived seeds until a validator accepts the layout. The engine's
// passes cover heights and ledges ([AssignHeights], [PlaceLiftLedge]),
// half walls ([PlaceLowWalls]), doors ([PlaceDoors]), vent networks
// ([CarveVents]), room themes ([AssignThemes]), and open sky
// ([AssignSky]); games append their own [Pass] values for items, markers,
// and hazards.
//
// A [Level] can equally be built by hand: [New] returns solid rock, and
// the Level satisfies [worldgen.Carver], so [worldgen.Generate] digs
// straight into it. Movement rules that heights introduce are answered by
// [Level.StepOK], and [LiftHeight] animates a [Lift] through its
// dwell-travel cycle.
//
// The [Tile] vocabulary is engine-owned and deliberately general — a
// [TileSwitch] is any wall-mounted interactable, a [TileCover] any hiding
// spot — while items, markers, hazards, themes, and every other gameplay
// meaning stays in each game.
package level

// Tile identifies what occupies one grid cell. The vocabulary is the
// engine's: games render and interpret tiles their own way but agree on
// the spatial meaning.
type Tile uint8

const (
	// TileFloor is open walkable ground.
	TileFloor Tile = iota
	// TileWall is solid rock, full height unless a wall-top height says
	// otherwise.
	TileWall
	// TileDoor is a doorway; whether it is currently passable is runtime
	// state owned by the game.
	TileDoor
	// TileVent is a crawlable tunnel cell inside a wall mass.
	TileVent
	// TileSwitch is a wall-mounted interactable: a switch, console, or
	// lever. It is not walkable.
	TileSwitch
	// TileCover is a spot an agent can hide in: a locker, alcove, or
	// crate shadow.
	TileCover
	// TileSpawn is the player's starting cell.
	TileSpawn
	// TileExit is the level's goal cell.
	TileExit
)

// Walkable reports whether agents may occupy the tile.
func (t Tile) Walkable() bool {
	switch t {
	case TileFloor, TileDoor, TileVent, TileCover, TileSpawn, TileExit:
		return true
	default:
		return false
	}
}

// Rune returns the tile's conventional map-dump character.
func (t Tile) Rune() rune {
	switch t {
	case TileWall:
		return '#'
	case TileDoor:
		return '+'
	case TileVent:
		return '~'
	case TileSwitch:
		return '!'
	case TileCover:
		return 'H'
	case TileSpawn:
		return 'S'
	case TileExit:
		return 'E'
	default:
		return '.'
	}
}
