// Package dbs reads Wizardry VI's data files. Bytes are decoded here, at the
// file boundary; the game sees typed values.
package dbs

import (
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

func le16(b []byte, o int) int { return int(b[o]) | int(b[o+1])<<8 }

// planarPixel reads a 4-plane pixel: bit p of the colour is bit (7-x%8) of
// planes[p][y*bpr + x/8].
func planarPixel(planes [4][]byte, bpr, x, y int) uint8 {
	i, m := y*bpr+x/8, byte(0x80>>(x%8))
	var c uint8
	for p := 0; p < 4; p++ {
		if i < len(planes[p]) && planes[p][i]&m != 0 {
			c |= 1 << p
		}
	}
	return c
}
