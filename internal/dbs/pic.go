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
	Glyphs [128][8][8]uint8
}

// LoadFont reads a 1024-byte mono or 4096-byte planar font.
func LoadFont(path string) (*Font, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f := &Font{}
	switch {
	case len(b) >= 4096:
		for g := 0; g < 128; g++ {
			gb := b[g*32:]
			pl := [4][]byte{gb[0:8], gb[8:16], gb[16:24], gb[24:32]}
			for y := 0; y < 8; y++ {
				for x := 0; x < 8; x++ {
					f.Glyphs[g][y][x] = planarPixel(pl, 1, x, y)
				}
			}
		}
	case len(b) >= 1024:
		f.Mono = true
		for g := 0; g < 128; g++ {
			for y := 0; y < 8; y++ {
				for x := 0; x < 8; x++ {
					f.Glyphs[g][y][x] = b[g*8+y] >> (7 - x) & 1
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
func LoadScreen(path string) ([320 * 200]uint8, error) {
	var px [320 * 200]uint8
	b, err := os.ReadFile(path)
	if err != nil {
		return px, err
	}
	if len(b) < 0x6000+8000 {
		return px, fmt.Errorf("%s: %d bytes, want 32768", path, len(b))
	}
	pl := [4][]byte{b[0:8000], b[0x2000 : 0x2000+8000], b[0x4000 : 0x4000+8000], b[0x6000 : 0x6000+8000]}
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			px[y*320+x] = planarPixel(pl, 40, x, y)
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

// UnRLE is op_ae's decompressor (EGA.DRV 0x1d25): the file is read in 4096-
// byte chunks and control bytes are taken only while the chunk index is below
// 0xfff. 0 ends; 1..0x7f copy that many literals; 0x80..0xff repeat the next
// byte -ctrl times.
func UnRLE(b []byte) []byte {
	var out []byte
	for base := 0; base < len(b); base += 0x1000 {
		chunk := b[base:min(base+0x1000, len(b))]
		at := func(i int) byte {
			if i < len(chunk) {
				return chunk[i]
			}
			return 0
		}
		for si := 0; si < 0xfff && si < len(chunk); {
			c := chunk[si]
			si++
			switch {
			case c == 0:
				return out
			case c < 0x80:
				for k := 0; k < int(c); k++ {
					out = append(out, at(si))
					si++
				}
			default:
				v := at(si)
				si++
				for k := 0; k < 256-int(c); k++ {
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
	if len(raw) < 24 {
		return nil, fmt.Errorf("%s: decompressed to %d bytes", path, len(raw))
	}
	p := &Pic{Raw: raw}
	n := le16(raw, 0) / 24
	for k := 0; k < n && k*24+24 <= len(raw); k++ {
		r := raw[k*24:]
		f := PicFrame{Offset: le16(r, 0), W: int(r[2]), H: int(r[3])}
		f.Present = make([]bool, f.W*f.H)
		f.Pix = make([]uint8, f.W*8*f.H*8)
		src := f.Offset
		for t := range f.Present {
			f.Present[t] = t < 160 && r[4+t/8]&(1<<(t%8)) != 0
			tx, ty := t%f.W*8, t/f.W*8
			if !f.Present[t] || src+32 > len(raw) {
				for y := 0; y < 8; y++ {
					for x := 0; x < 8; x++ {
						f.Pix[(ty+y)*f.W*8+tx+x] = 15
					}
				}
				continue
			}
			tb := raw[src:]
			pl := [4][]byte{tb[0:8], tb[8:16], tb[16:24], tb[24:32]}
			for y := 0; y < 8; y++ {
				for x := 0; x < 8; x++ {
					f.Pix[(ty+y)*f.W*8+tx+x] = planarPixel(pl, 1, x, y)
				}
			}
			src += 32
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

// LoadPortraits reads every whole 288-byte portrait in the file.
func LoadPortraits(path string) ([]Portrait, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []Portrait
	for o := 0; o+288 <= len(b); o += 288 {
		p := Portrait{W: 24, H: 24, Pix: make([]uint8, 24*24)}
		for t := 0; t < 9; t++ {
			tb := b[o+t*32:]
			pl := [4][]byte{tb[0:8], tb[8:16], tb[16:24], tb[24:32]}
			for y := 0; y < 8; y++ {
				for x := 0; x < 8; x++ {
					p.Pix[(t/3*8+y)*24+t%3*8+x] = planarPixel(pl, 1, x, y)
				}
			}
		}
		out = append(out, p)
	}
	return out, nil
}
