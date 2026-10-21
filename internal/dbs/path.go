// Package dbs reads Wizardry VI's data files. Bytes are decoded here, at the
// file boundary; the game sees typed values.
package dbs

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
)

// Path returns dir/name, matching name case-insensitively (DOS names on a
// case-sensitive disk). If nothing matches it returns dir/name unchanged.
func Path(dir, name string) string {
	p := filepath.Join(dir, name)
	if _, err := os.Stat(p); err == nil {
		return p
	}
	ents, _ := os.ReadDir(dir)
	for _, e := range ents {
		if strings.EqualFold(e.Name(), name) {
			return filepath.Join(dir, e.Name())
		}
	}
	return p
}

func readFile(dir, name string) ([]byte, error) { return os.ReadFile(Path(dir, name)) }

func le16(b []byte, o int) int { return int(binary.LittleEndian.Uint16(b[o:])) }

const (
	planeCount    = 4 // EGA bit planes, one bit of the 16-colour index each
	pixelsPerByte = 8 // in one plane, leftmost pixel in the high bit
	tileSize      = 8 // glyphs and picture tiles are 8x8 pixels
	// planarTileBytes is one 8x8 tile: its planes one after the other.
	planarTileBytes = planeCount * tileSize
)

// planarPixel reads a 4-plane pixel: bit p of the colour is bit (7-x%8) of
// planes[p][y*bpr + x/8].
func planarPixel(planes [planeCount][]byte, bpr, x, y int) uint8 {
	i, m := y*bpr+x/pixelsPerByte, byte(1)<<(pixelsPerByte-1-x%pixelsPerByte)
	var c uint8
	for p := 0; p < planeCount; p++ {
		if i < len(planes[p]) && planes[p][i]&m != 0 {
			c |= 1 << p
		}
	}
	return c
}
