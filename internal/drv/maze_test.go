package drv

import (
	"testing"

	"github.com/retro-ware/wiz6-go/internal/dbs"
)

// A mirrored draw is the plain draw flipped, when both use the same record and the
// record spans the whole tile.
func TestMazeTileMirror(t *testing.T) {
	m, err := dbs.LoadMazeData(data + "MAZEDATA.EGA")
	if err != nil {
		t.Skip("no data:", err)
	}
	LoadMaze(m)
	for r, d := range m.Draws {
		tl := m.Tiles[d.Tile]
		if d.Skip != 0 || int(d.Width) != tl.W || d.X < 0 || int(d.X)+tl.W > Cols || int(d.Y)+tl.H > ScreenHeight {
			continue
		}
		WorkBuffer = [ScreenHeight][ScreenWidth]uint8{}
		MazeTile(r, 0, -1)
		plain := WorkBuffer
		WorkBuffer = [ScreenHeight][ScreenWidth]uint8{}
		MazeTile(r, 0, r)
		n := 0
		x0, x1 := int(d.X)*8, (int(d.X)+tl.W)*8-1
		for y := int(d.Y); y < int(d.Y)+tl.H; y++ {
			for x := x0; x <= x1; x++ {
				if plain[y][x] != WorkBuffer[y][x0+x1-x] {
					t.Fatalf("record %d: (%d,%d) %d, mirrored %d", r, x, y, plain[y][x], WorkBuffer[y][x0+x1-x])
				}
				if plain[y][x] != 0 {
					n++
				}
			}
		}
		if n == 0 {
			continue
		}
		return
	}
	t.Fatal("no full-tile record with pixels")
}
