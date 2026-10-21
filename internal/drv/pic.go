package drv

// The picture slot area (cs:[0x169], 0x1c00 paragraphs). Slot n starts at slotBase[n]
// paragraphs (driver table cs:017a). A picture larger than its slot runs on into the
// next ones, as in the original, and the pristine full-screen copy occupies slot area 5
// onwards (cs:[0x184]).
var (
	slots    [0x1c000]byte
	slotBase = [10]int{0x0000, 0x0180, 0x0340, 0x0600, 0x0700, 0x0d00, 0x1300, 0x1900, 0x1a00, 0x1b00}
	compose  [0x1400]byte // the scratch buffer draw_text builds the picture in (cs:[0x16d])
)

const savedPara = 0x0d00 // slotBase[5]

// LoadPic RLE-decompresses a picture file (MONnn.PIC, CREDITS.PIC) into slot n (driver
// 1d25, drv_op_ae). The file is read in 4096-byte chunks; control bytes are taken only
// while fewer than 0xfff bytes of the chunk are used: 0 ends, 1..7f copies that many
// literal bytes, 80..ff repeats the next byte -ctrl times.
func LoadPic(n int, file []byte) {
	di := slotBase[n] * 16
	put := func(b byte) {
		if di < len(slots) {
			slots[di] = b
		}
		di++
	}
	for chunk := 0; chunk < len(file); chunk += 0x1000 {
		buf := make([]byte, 0x1000+0x100) // a control byte at 0xffe may read past the chunk
		copy(buf, file[chunk:])
		for si := 0; si < 0xfff; {
			c := buf[si]
			si++
			switch {
			case c == 0:
				return
			case c < 0x80:
				for range int(c) {
					put(buf[si])
					si++
				}
			default:
				v := buf[si]
				si++
				for range 256 - int(c) {
					put(v)
				}
			}
		}
	}
}

// LoadScreen loads a 32768-byte full-screen picture (TITLEPAG.EGA, DRAGONSC.EGA, ...) into
// the pristine copy in slot area 5 and copies it to the working buffer (driver 06f0,
// drv_op_b6). With file nil (fd -1 in the C) it only restores the working buffer from the
// pristine copy, whatever slot area 5 holds now.
func LoadScreen(file []byte) {
	base := savedPara * 16
	if file != nil {
		n := copy(slots[base:base+0x8000], file)
		clear(slots[base+n : base+0x8000])
	}
	p := slots[base:]
	for y := range H {
		for x := range W {
			i := y*40 + x>>3
			m := byte(0x80) >> (x & 7)
			var c uint8
			for pl := range 4 {
				if p[pl*0x2000+i]&m != 0 {
					c |= 1 << pl
				}
			}
			Back[y][x] = c
		}
	}
}

// DrawFrames draws the frames listed in frames (1-based frame numbers, ended by 0 or the
// slice's end) of the picture in slot n, overlaid, at pixel (x, y) of the working buffer
// (driver 1d94, drv_draw_text). All frames take the first frame's size. Columns left of
// clipL and from clipR on stay clear, as do rows from 144 on. flags bit 0 mirrors the
// picture left-right, bit 1 flips it upside down. With toScreen the result goes to the
// screen over the working buffer's pixels and the working buffer stays as it was.
func DrawFrames(n, x, y, clipL, clipR, flags int, toScreen bool, frames []byte) {
	base := slotBase[n] * 16
	rec := func(f byte) int { return base + (int(f)-1)*24 }
	if len(frames) == 0 || frames[0] == 0 {
		return
	}
	r0 := rec(frames[0])
	tw, th := int(slots[r0+2]), int(slots[r0+3])
	if tw == 0 && th == 0 {
		return
	}
	pw, ph := tw*8, th*8
	if x+pw <= clipL || x >= clipR {
		return
	}
	// The composite, one colour per pixel; Transparent where no frame has a pixel.
	img := make([]uint8, pw*ph)
	for i := range img {
		img[i] = Transparent
	}
	for _, f := range frames {
		if f == 0 {
			break
		}
		r := rec(f)
		src := base + (int(slots[r]) | int(slots[r+1])<<8)
		bit := 0
		for ty := range th {
			for tx := range tw {
				present := slots[r+4+bit>>3]&(1<<(bit&7)) != 0
				bit++
				if !present {
					continue
				}
				for py := range 8 {
					for px := range 8 {
						c := tilePixel(src, px, py)
						if c != Transparent {
							img[(ty*8+py)*pw+tx*8+px] = c
						}
					}
				}
				src += 32
			}
		}
	}
	at := func(ix, iy int) uint8 {
		if flags&1 != 0 {
			ix = pw - 1 - ix
		}
		if flags&2 != 0 {
			iy = ph - 1 - iy
		}
		return img[iy*pw+ix]
	}
	for iy := range ph {
		sy := y + iy
		if sy < 0 || sy >= 144 {
			continue
		}
		for ix := range pw {
			sx := x + ix
			if sx < 0 || sx >= W {
				continue
			}
			c := at(ix, iy)
			if sx < clipL || sx >= clipR {
				c = Transparent
			}
			if c == Transparent {
				if toScreen {
					Screen[sy][sx] = Back[sy][sx]
				}
				continue
			}
			if toScreen {
				Screen[sy][sx] = c
			} else {
				Back[sy][sx] = c
			}
		}
	}
	if toScreen {
		dirty = true
		Present()
	}
}

// tilePixel is pixel (x, y) of the 32-byte planar tile at slot offset src.
func tilePixel(src, x, y int) uint8 {
	var c uint8
	m := byte(0x80) >> x
	for pl := range 4 {
		if src+pl*8+y < len(slots) && slots[src+pl*8+y]&m != 0 {
			c |= 1 << pl
		}
	}
	return c
}
