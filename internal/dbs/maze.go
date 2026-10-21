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

// mazeTileRec is a MAZEDATA.EGA tile record as stored.
type mazeTileRec struct {
	Seg  uint16 // paragraphs
	Sub  uint8
	W, H uint8
}

// mazeCounts is the MAZEDATA.EGA header.
type mazeCounts struct {
	Tiles, Draws uint16
}

const paragraphSize = 16 // a real-mode segment step

// LoadMazeData reads MAZEDATA.EGA.
func LoadMazeData(path string) (*MazeData, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	mazeCountsSize := Size[mazeCounts]()
	if len(b) < mazeCountsSize {
		return nil, fmt.Errorf("%s: too short", path)
	}
	counts, _ := Decode[mazeCounts](b[:mazeCountsSize])
	n, n2 := int(counts.Tiles), int(counts.Draws)
	tileSz, drawSz := Size[mazeTileRec](), Size[MazeDraw]()
	hdr := mazeCountsSize + tileSz*n + drawSz*n2
	if hdr > len(b) {
		return nil, fmt.Errorf("%s: header %d past end", path, hdr)
	}
	m := &MazeData{Raw: b}
	for i := 0; i < n; i++ {
		at := mazeCountsSize + tileSz*i
		r, _ := Decode[mazeTileRec](b[at : at+tileSz])
		t := MazeTile{W: int(r.W), H: int(r.H), Off: hdr + int(r.Seg)*paragraphSize + int(r.Sub)}
		width := t.W * pixelsPerByte
		t.Pix = make([]uint8, width*t.H)
		ps := t.W * t.H
		if t.Off+planeCount*ps <= len(b) {
			var pl [planeCount][]byte
			for p := range pl {
				pl[p] = b[t.Off+p*ps:]
			}
			for y := 0; y < t.H; y++ {
				for x := 0; x < width; x++ {
					t.Pix[y*width+x] = planarPixel(pl, t.W, x, y)
				}
			}
		}
		m.Tiles = append(m.Tiles, t)
	}
	for i := 0; i < n2; i++ {
		at := mazeCountsSize + tileSz*n + drawSz*i
		d, _ := Decode[MazeDraw](b[at : at+drawSz])
		m.Draws = append(m.Draws, d)
	}
	return m, nil
}
