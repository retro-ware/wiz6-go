package drv

import "math/bits"

// MazeTile draws MAZEDATA draw record draw into the working buffer (driver 0b93,
// drv_op_c6). The record names a tile and where it goes: Width bytes of each of the
// tile's rows, from byte Skip of the row on, at byte column X+Skip of pixel row Y. With
// or set the tile's bits are ORed into the buffer (colour 0 lets what is below show),
// otherwise they replace it. With mirror >= 0 the tile of record draw is drawn mirrored
// left to right, placed and clipped by record mirror instead.
func MazeTile(draw, or, mirror int) {
	if mazeData == nil || draw < 0 || draw >= len(mazeData.Draws) {
		return
	}
	d := mazeData.Draws[draw]
	if int(d.Tile) >= len(mazeData.Tiles) {
		return
	}
	t := mazeData.Tiles[d.Tile]
	flip := mirror != -1
	if flip {
		if mirror < 0 || mirror >= len(mazeData.Draws) {
			return
		}
		d = mazeData.Draws[mirror]
	}
	width := int(d.Width)
	if width == 0 {
		return
	}
	dst := int(d.X) + int(d.Skip) + int(d.Y)*Cols
	planeSize := t.W * t.H
	src := t.Off + int(d.Skip)
	if flip {
		src = t.Off + t.W - 1 - int(d.Skip)
	}
	for range t.H {
		for i := range width {
			s := src + i
			if flip {
				s = src - i
			}
			var b [planeCount]byte
			for p := range b {
				if k := s + p*planeSize; k >= 0 && k < len(mazeData.Raw) {
					b[p] = mazeData.Raw[k]
					if flip {
						b[p] = bits.Reverse8(b[p])
					}
				}
			}
			putPlanarByte(dst+i, b, or != 0)
		}
		dst += Cols
		src += t.W
	}
}

// putPlanarByte writes the four plane bytes of byte offset o (40 a row) of the working
// buffer: 8 pixels, bit 7 leftmost, plane p is bit p of the colour. Offsets past the
// picture are memory the screen does not show.
func putPlanarByte(o int, b [planeCount]byte, or bool) {
	if o < 0 || o >= Cols*ScreenHeight {
		return
	}
	row := &WorkBuffer[o/Cols]
	x0 := o % Cols * CellSize
	for x := range CellSize {
		var c uint8
		for p := range planeCount {
			if b[p]&pixelBit(x) != 0 {
				c |= 1 << p
			}
		}
		if or {
			c |= row[x0+x]
		}
		row[x0+x] = c
	}
}
