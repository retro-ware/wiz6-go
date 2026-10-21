package dbs

import (
	"fmt"
	"os"
)

// MazeTile is a MAZEDATA tile as EGA.DRV load_from_file (0x731) relocates it.
// The pixels live at file offset Off = header size + Seg*16 + Sub, as 4
// sequential planes of W*H bytes (W bytes per row). Pix is W*8 x H indices.
type MazeTile struct {
	W, H int // W in bytes (8 pixels), H in rows
	Off  int
	Pix  []uint8
}

// MazeDraw is one 5-byte display record that op_c6 (0xb93) draws: tile Tile
// at byte column X (signed) and row Y, skipping Skip bytes of each tile row
// (added to source and destination) and copying Width bytes per row for all
// H rows of the tile. With a second record, op_c6 mirrors the tile instead.
type MazeDraw struct {
	Tile  uint8
	X     int8
	Y     uint8
	Skip  uint8
	Width uint8
}

// MazeData is MAZEDATA.EGA: word N tiles, word N2 draws, N*5 tile records
// {u16 seg, u8 sub, u8 w, u8 h}, N2*5 draw records, then pixel data. Raw
// keeps the file so a copy that overruns a tile reads what the driver reads.
type MazeData struct {
	Tiles []MazeTile
	Draws []MazeDraw
	Raw   []byte
}

// LoadMazeData reads MAZEDATA.EGA.
func LoadMazeData(path string) (*MazeData, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) < 4 {
		return nil, fmt.Errorf("%s: too short", path)
	}
	n, n2 := le16(b, 0), le16(b, 2)
	hdr := 4 + 5*(n+n2)
	if hdr > len(b) {
		return nil, fmt.Errorf("%s: header %d past end", path, hdr)
	}
	m := &MazeData{Raw: b}
	for i := 0; i < n; i++ {
		r := b[4+5*i:]
		t := MazeTile{W: int(r[3]), H: int(r[4]), Off: hdr + le16(r, 0)*16 + int(r[2])}
		t.Pix = make([]uint8, t.W*8*t.H)
		ps := t.W * t.H
		if t.Off+4*ps <= len(b) {
			pl := [4][]byte{b[t.Off:], b[t.Off+ps:], b[t.Off+2*ps:], b[t.Off+3*ps:]}
			for y := 0; y < t.H; y++ {
				for x := 0; x < t.W*8; x++ {
					t.Pix[y*t.W*8+x] = planarPixel(pl, t.W, x, y)
				}
			}
		}
		m.Tiles = append(m.Tiles, t)
	}
	for i := 0; i < n2; i++ {
		r := b[4+5*n+5*i:]
		m.Draws = append(m.Draws, MazeDraw{r[0], int8(r[1]), r[2], r[3], r[4]})
	}
	return m, nil
}
