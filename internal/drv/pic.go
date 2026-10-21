package drv

import "encoding/binary"

// The picture slot area (cs:[0x169], 0x1c00 paragraphs). Slot n starts at slotStartParagraph[n]
// paragraphs (driver table cs:017a). A picture larger than its slot runs on into the
// next ones, as in the original, and the pristine full-screen copy occupies slot area 5
// onwards (cs:[0x184]).
const (
	paragraphSize      = 16   // bytes in a real-mode paragraph
	slotAreaParagraphs = 7168 // 0x1c00
	// pristineScreenSlot is the slot area the pristine full-screen copy starts at.
	pristineScreenSlot = 5
)

var (
	slotArea [slotAreaParagraphs * paragraphSize]byte
	// slotStartParagraph is 0000 0180 0340 0600 0700 0d00 1300 1900 1a00 1b00 in hex.
	slotStartParagraph = [10]int{0, 384, 832, 1536, 1792, 3328, 4864, 6400, 6656, 6912}
)

// The RLE picture files (drv_op_ae).
const (
	rleChunkSize = 4096
	// rleControlLimit: control bytes are read only below this chunk index (0xfff).
	rleControlLimit = rleChunkSize - 1
	// rleChunkSlack covers a control byte near the chunk's end reading past it.
	rleChunkSlack  = 256
	rleEnd         = 0   // control byte: the picture ends
	rleRepeatFirst = 128 // control bytes from here on repeat the next byte 256-ctrl times
	rleRepeatBase  = 256
)

// Full-screen pictures (drv_op_b6): four planes, each a plane byte per cell row of the screen.
const (
	screenPlaneStride = 8192 // 0x2000: where the next plane starts
	screenPictureSize = planeCount * screenPlaneStride
)

// The frames of a slot's picture (drv_draw_text).
const (
	// frameRecordSize is a frame record's size: tile data offset (2 bytes, little-endian, from
	// the slot's start), width and height in tiles, then a bitmap of the tiles present, bit 0
	// of each byte first.
	frameRecordSize      = 24
	frameTilesWideOffset = 2
	frameTilesHighOffset = 3
	frameBitmapOffset    = 4
	// tileSize is a present tile's size: planeCount planes of CellSize row bytes.
	tileSize = planeCount * CellSize
	// drawFramesBottom is the first pixel row DrawFrames leaves clear.
	drawFramesBottom = 144
	// DrawFrames flags.
	mirrorLeftRight = 1
	flipUpsideDown  = 2
)

// LoadPic RLE-decompresses a picture file (MONnn.PIC, CREDITS.PIC) into slot n (driver
// 1d25, drv_op_ae). The file is read in 4096-byte chunks; control bytes are taken only
// while fewer than 0xfff bytes of the chunk are used: 0 ends, 1..7f copies that many
// literal bytes, 80..ff repeats the next byte -ctrl times.
func LoadPic(n int, file []byte) {
	di := slotStartParagraph[n] * paragraphSize
	put := func(b byte) {
		if di < len(slotArea) {
			slotArea[di] = b
		}
		di++
	}
	for chunk := 0; chunk < len(file); chunk += rleChunkSize {
		buf := make([]byte, rleChunkSize+rleChunkSlack) // a control byte at 0xffe may read past the chunk
		copy(buf, file[chunk:])
		for si := 0; si < rleControlLimit; {
			c := buf[si]
			si++
			switch {
			case c == rleEnd:
				return
			case c < rleRepeatFirst:
				for range int(c) {
					put(buf[si])
					si++
				}
			default:
				v := buf[si]
				si++
				for range rleRepeatBase - int(c) {
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
	base := slotStartParagraph[pristineScreenSlot] * paragraphSize
	if file != nil {
		n := copy(slotArea[base:base+screenPictureSize], file)
		clear(slotArea[base+n : base+screenPictureSize])
	}
	p := slotArea[base:]
	for y := range ScreenHeight {
		for x := range ScreenWidth {
			i := y*Cols + x/CellSize
			m := pixelBit(x % CellSize)
			var c uint8
			for pl := range planeCount {
				if p[pl*screenPlaneStride+i]&m != 0 {
					c |= 1 << pl
				}
			}
			WorkBuffer[y][x] = c
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
	base := slotStartParagraph[n] * paragraphSize
	rec := func(f byte) int { return base + (int(f)-1)*frameRecordSize }
	if len(frames) == 0 || frames[0] == 0 {
		return
	}
	r0 := rec(frames[0])
	tw, th := int(slotArea[r0+frameTilesWideOffset]), int(slotArea[r0+frameTilesHighOffset])
	if tw == 0 && th == 0 {
		return
	}
	pw, ph := tw*CellSize, th*CellSize
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
		src := base + int(binary.LittleEndian.Uint16(slotArea[r:]))
		bit := 0
		for ty := range th {
			for tx := range tw {
				present := slotArea[r+frameBitmapOffset+bit/8]&(1<<(bit%8)) != 0
				bit++
				if !present {
					continue
				}
				for py := range CellSize {
					for px := range CellSize {
						c := tilePixel(src, px, py)
						if c != Transparent {
							img[(ty*CellSize+py)*pw+tx*CellSize+px] = c
						}
					}
				}
				src += tileSize
			}
		}
	}
	at := func(ix, iy int) uint8 {
		if flags&mirrorLeftRight != 0 {
			ix = pw - 1 - ix
		}
		if flags&flipUpsideDown != 0 {
			iy = ph - 1 - iy
		}
		return img[iy*pw+ix]
	}
	for iy := range ph {
		sy := y + iy
		if sy < 0 || sy >= drawFramesBottom {
			continue
		}
		for ix := range pw {
			sx := x + ix
			if sx < 0 || sx >= ScreenWidth {
				continue
			}
			c := at(ix, iy)
			if sx < clipL || sx >= clipR {
				c = Transparent
			}
			if c == Transparent {
				if toScreen {
					Screen[sy][sx] = WorkBuffer[sy][sx]
				}
				continue
			}
			if toScreen {
				Screen[sy][sx] = c
			} else {
				WorkBuffer[sy][sx] = c
			}
		}
	}
	if toScreen {
		screenChanged = true
		Present()
	}
}

// tilePixel is pixel (x, y) of the 32-byte planar tile at slot offset src.
func tilePixel(src, x, y int) uint8 {
	var c uint8
	m := pixelBit(x)
	for pl := range planeCount {
		if k := src + pl*CellSize + y; k < len(slotArea) && slotArea[k]&m != 0 {
			c |= 1 << pl
		}
	}
	return c
}
