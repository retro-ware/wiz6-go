package dbs

import (
	"fmt"
	"os"
)

// Font is a WFONTn.EGA font as load_font (ROOT 0x21bb) keeps it: the raw file
// read into the driver's font segment (1024 bytes for font 0, 4096 for 1-4 on
// EGA; CGA/Tandy sizes differ and are not handled). Mono glyphs (font 0) hold
// 0/1 bits; colour glyphs hold 4-plane indices 0..15. Glyphs[g][y][x].
type Font struct {
	Mono   bool
	Glyphs [glyphCount][tileSize][tileSize]uint8
}

const (
	glyphCount     = 128
	monoFontSize   = glyphCount * tileSize        // 1 bit per pixel
	planarFontSize = glyphCount * planarTileBytes // 4 planes
)

// Screen geometry of LoadScreen's full-screen pictures.
const (
	ScreenWidth     = 320
	ScreenHeight    = 200
	screenRowBytes  = ScreenWidth / pixelsPerByte
	screenPlaneUsed = screenRowBytes * ScreenHeight // 8000
	screenPlaneSlot = 8192                          // planes start 8K apart
)

// tilePlanes splits one planar 8x8 tile into its planes.
func tilePlanes(tile []byte) [planeCount][]byte {
	var pl [planeCount][]byte
	for p := range pl {
		pl[p] = tile[p*tileSize : (p+1)*tileSize]
	}
	return pl
}

// LoadFont reads a 1024-byte mono or 4096-byte planar font.
func LoadFont(path string) (*Font, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f := &Font{}
	switch {
	case len(b) >= planarFontSize:
		for g := 0; g < glyphCount; g++ {
			pl := tilePlanes(b[g*planarTileBytes:])
			for y := 0; y < tileSize; y++ {
				for x := 0; x < tileSize; x++ {
					f.Glyphs[g][y][x] = planarPixel(pl, 1, x, y)
				}
			}
		}
	case len(b) >= monoFontSize:
		f.Mono = true
		for g := 0; g < glyphCount; g++ {
			for y := 0; y < tileSize; y++ {
				for x := 0; x < tileSize; x++ {
					f.Glyphs[g][y][x] = b[g*tileSize+y] >> (pixelsPerByte - 1 - x) & 1
				}
			}
		}
	default:
		return nil, fmt.Errorf("%s: %d bytes is not a font", path, len(b))
	}
	return f, nil
}

// LoadScreen reads a full-screen picture (TITLEPAG/DRAGONSC/GRAVEYRD.EGA):
// four 8192-byte plane slots, 8000 bytes used, 40 bytes per row. EGA.DRV
// op_b6 copies the file's 32K as is.
func LoadScreen(path string) ([ScreenWidth * ScreenHeight]uint8, error) {
	var px [ScreenWidth * ScreenHeight]uint8
	b, err := os.ReadFile(path)
	if err != nil {
		return px, err
	}
	if len(b) < (planeCount-1)*screenPlaneSlot+screenPlaneUsed {
		return px, fmt.Errorf("%s: %d bytes, want %d", path, len(b), planeCount*screenPlaneSlot)
	}
	var pl [planeCount][]byte
	for p := range pl {
		pl[p] = b[p*screenPlaneSlot : p*screenPlaneSlot+screenPlaneUsed]
	}
	for y := 0; y < ScreenHeight; y++ {
		for x := 0; x < ScreenWidth; x++ {
			px[y*ScreenWidth+x] = planarPixel(pl, screenRowBytes, x, y)
		}
	}
	return px, nil
}

// PicFrame is one 24-byte frame record of a decompressed picture. W, H are in
// 8-pixel tiles; Present marks the tiles stored (row-major, mask LSB-first).
// Absent tiles and colour 15 are transparent. Pix is W*8 x H*8, absent tiles 15.
type PicFrame struct {
	Offset  int // record's data offset in Raw
	W, H    int
	Present []bool
	Pix     []uint8
}

// Pic is a .PIC file (MONnn.PIC, CREDITS.PIC) after EGA.DRV op_ae's RLE
// decompression. Raw begins with the frame records, record k at k*24; their
// count is record 0's offset / 24. draw_text frame n (1-based) is Frames[n-1].
type Pic struct {
	Frames []PicFrame
	Raw    []byte
}

// picFrameRec is a frame record as stored.
type picFrameRec struct {
	Offset  uint16
	W, H    uint8
	Present [20]uint8 // tile bits, LSB first
}

const (
	picMaxTiles       = len(picFrameRec{}.Present) * bitsPerByte
	transparentColour = 15

	rleChunkSize = 4096 // op_ae reads the file in chunks this size
	rleRunFlag   = 128  // control bytes from this up are runs
)

// UnRLE is op_ae's decompressor (EGA.DRV 0x1d25): the file is read in 4096-
// byte chunks and control bytes are taken only while the chunk index is below
// 0xfff. 0 ends; 1..0x7f copy that many literals; 0x80..0xff repeat the next
// byte -ctrl times.
func UnRLE(b []byte) []byte {
	var out []byte
	for base := 0; base < len(b); base += rleChunkSize {
		chunk := b[base:min(base+rleChunkSize, len(b))]
		at := func(i int) byte {
			if i < len(chunk) {
				return chunk[i]
			}
			return 0
		}
		for si := 0; si < rleChunkSize-1 && si < len(chunk); {
			c := chunk[si]
			si++
			switch {
			case c == 0:
				return out
			case c < rleRunFlag:
				for k := 0; k < int(c); k++ {
					out = append(out, at(si))
					si++
				}
			default:
				v := at(si)
				si++
				for k := 0; k < -int(int8(c)); k++ {
					out = append(out, v)
				}
			}
		}
	}
	return out
}

// LoadPic reads and decodes a .PIC file.
func LoadPic(path string) (*Pic, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	raw := UnRLE(b)
	recSize := Size[picFrameRec]()
	if len(raw) < recSize {
		return nil, fmt.Errorf("%s: decompressed to %d bytes", path, len(raw))
	}
	p := &Pic{Raw: raw}
	n := le16(raw, 0) / recSize
	for k := 0; k < n && (k+1)*recSize <= len(raw); k++ {
		r, _ := Decode[picFrameRec](raw[k*recSize : (k+1)*recSize])
		f := PicFrame{Offset: int(r.Offset), W: int(r.W), H: int(r.H)}
		f.Present = make([]bool, f.W*f.H)
		width := f.W * tileSize
		f.Pix = make([]uint8, width*f.H*tileSize)
		src := f.Offset
		for t := range f.Present {
			f.Present[t] = t < picMaxTiles && r.Present[t/bitsPerByte]&(1<<(t%bitsPerByte)) != 0
			tx, ty := t%f.W*tileSize, t/f.W*tileSize
			if !f.Present[t] || src+planarTileBytes > len(raw) {
				for y := 0; y < tileSize; y++ {
					for x := 0; x < tileSize; x++ {
						f.Pix[(ty+y)*width+tx+x] = transparentColour
					}
				}
				continue
			}
			pl := tilePlanes(raw[src:])
			for y := 0; y < tileSize; y++ {
				for x := 0; x < tileSize; x++ {
					f.Pix[(ty+y)*width+tx+x] = planarPixel(pl, 1, x, y)
				}
			}
			src += planarTileBytes
		}
		p.Frames = append(p.Frames, f)
	}
	return p, nil
}

// Portrait is one 24x24 WPORTn.EGA portrait: 3x3 planar 8x8 tiles, 288 bytes.
type Portrait struct {
	W, H int
	Pix  []uint8
}

const (
	portraitTiles = 3 // tiles per side
	portraitSide  = portraitTiles * tileSize
	portraitBytes = portraitTiles * portraitTiles * planarTileBytes // 288
)

// LoadPortraits reads every whole 288-byte portrait in the file.
func LoadPortraits(path string) ([]Portrait, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []Portrait
	for o := 0; o+portraitBytes <= len(b); o += portraitBytes {
		p := Portrait{W: portraitSide, H: portraitSide, Pix: make([]uint8, portraitSide*portraitSide)}
		for t := 0; t < portraitTiles*portraitTiles; t++ {
			pl := tilePlanes(b[o+t*planarTileBytes:])
			tx, ty := t%portraitTiles*tileSize, t/portraitTiles*tileSize
			for y := 0; y < tileSize; y++ {
				for x := 0; x < tileSize; x++ {
					p.Pix[(ty+y)*portraitSide+tx+x] = planarPixel(pl, 1, x, y)
				}
			}
		}
		out = append(out, p)
	}
	return out, nil
}
