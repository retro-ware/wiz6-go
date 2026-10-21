// Package drv reimplements EGA.DRV, the original's graphics driver, on a 320x200 frame of
// 16 EGA colours. It keeps what the driver keeps (the screen, the working picture buffer,
// the pristine copy of the last full-screen picture, picture slots, fonts and the maze
// tiles) as typed Go values and does what each driver entry point does to them. There are
// no planes, ports or segments: a pixel is a colour index 0..15.
//
// Entry points, by their far-pointer slot in the root (cs:1b8a + 4*n) and driver address:
//
//	1b8a 029a  Glyph       mono glyph of font 0 in a text cell (put_char, refresh_window)
//	1b8e 053a  GlyphHi     colour glyph of font 1..4 in a text cell (put_char_hi, borders)
//	1b92 065f  FillCell    a text cell no window owns, in colour 8 (refresh_all_windows)
//	1b96 0fae  MouseInit   reset the mouse, show the pointer (drv_op_96)
//	1b9a 100a  MouseShutdown
//	1b9e 10f7  PollClick   the click region under the pointer (drv_poll_click)
//	1ba2 10e1  hide pointer / 1ba6 10ab show pointer (around every screen write)
//	1baa 0e2f  CopyCell    a text cell of a picture window, from the working buffer
//	1bae 1d25  LoadPic     RLE picture into a slot (drv_op_ae)
//	1bb2 1d94  DrawFrames  frames of a slot's picture into the working buffer (drv_draw_text)
//	1bb6 06f0  LoadScreen  full-screen picture into the pristine copy and the working buffer (drv_op_b6)
//	1bba 0731  LoadMaze    MAZEDATA tiles (drv_load_from_file)
//	1bbe 0835  FillBack    a window's rectangle of the working buffer (drv_op_be)
//	1bc2 08b3  ShowWindow  a window's rectangle, working buffer to screen (drv_op_c2)
//	1bc6 0b93  MazeTile    one maze wall tile into the working buffer (drv_op_c6)
//	1bca 1a2e  ClearClick  forget a latched click (flush_keys)
package drv

import (
	"github.com/retro-ware/wiz6-go/internal/dbs"
	"github.com/retro-ware/wiz6-go/internal/host"
)

// Screen size in pixels and in 8x8 text cells.
const (
	W, H       = 320, 200
	Cols, Rows = 40, 25
)

// Transparent is the colour a picture pixel with all four planes set has: draw_text and
// the maze blits leave the pixel below.
const Transparent = 15

// A Rect is a text-cell rectangle (a window's x, y, width, height).
type Rect struct{ X, Y, W, H int }

var (
	// Screen is the EGA screen (segment A000).
	Screen [H][W]uint8
	// Back is the working picture buffer (cs:[0x14d]): picture windows show it, draw_text
	// and the maze draw into it.
	Back  [H][W]uint8
	mono  [128][8]uint8       // font 0: one bit per pixel, bit 7 leftmost
	fonts [5][128][8][8]uint8 // fonts 1..4: colour indices
	dirty bool
	frame [W * H]byte
)

// Palette is the EGA palette the root sets for mode 0Dh (int 10h ax=1002h, root 1e43),
// as RGB with 8 bits each. Colour 15 is cyan; in pictures it means transparent.
var Palette = [16][3]uint8{
	{0, 0, 0}, {255, 255, 255}, {85, 85, 255}, {255, 85, 255},
	{255, 85, 85}, {255, 255, 85}, {85, 255, 85}, {85, 255, 255},
	{85, 85, 85}, {170, 170, 170}, {0, 0, 170}, {170, 0, 170},
	{170, 0, 0}, {170, 85, 0}, {0, 170, 0}, {0, 170, 170},
}

// Init sets the host palette and clears the screen, as the root's mode set does.
func Init() {
	cs := make([]host.Color, 16)
	for i, c := range Palette {
		cs[i] = host.Color{R: c[0] >> 2, G: c[1] >> 2, B: c[2] >> 2}
	}
	host.SetPalette(0, cs)
	Screen = [H][W]uint8{}
	dirty = true
	Present()
}

// SetMonoFont loads font 0 (WFONT0): row bytes, bit 7 the leftmost pixel.
func SetMonoFont(g [128][8]uint8) { mono = g }

// SetFont loads colour font n (1..4): 8x8 colour indices per glyph.
func SetFont(n int, g [128][8][8]uint8) { fonts[n] = g }

// SetGlyphs replaces count glyphs of colour font n from first on (root f2256).
func SetGlyphs(n, first int, g [][8][8]uint8) { copy(fonts[n][first:], g) }

// Present hands the screen to the host if it changed since the last call.
func Present() {
	if !dirty {
		return
	}
	dirty = false
	for y := range Screen {
		copy(frame[y*W:], Screen[y][:])
	}
	drawPointer(frame[:])
	host.Present(frame[:], W, H)
}

// Glyph draws glyph ch of font 0 in text cell (col, row) in colour fg (driver 029a). With
// inverse the glyph's clear pixels take fg and its set pixels colour 0.
func Glyph(col, row int, ch uint8, fg uint8, inverse bool) {
	g := mono[ch&0x7f]
	for y := range 8 {
		for x := range 8 {
			on := g[y]&(0x80>>x) != 0
			c := uint8(0)
			if on != inverse {
				c = fg & 15
			}
			Screen[row*8+y][col*8+x] = c
		}
	}
	dirty = true
}

// GlyphHi draws glyph ch of colour font n (1..4) in text cell (col, row) (driver 053a).
func GlyphHi(col, row int, ch uint8, n int) {
	g := &fonts[n][ch&0x7f]
	for y := range 8 {
		copy(Screen[row*8+y][col*8:col*8+8], g[y][:])
	}
	dirty = true
}

// FillCell paints text cell (col, row) in colour 8 (driver 065f, write mode 2).
func FillCell(col, row int) {
	for y := range 8 {
		for x := range 8 {
			Screen[row*8+y][col*8+x] = 8
		}
	}
	dirty = true
}

// CopyCell copies text cell (col, row) from the working buffer to the screen (driver 0e2f).
func CopyCell(col, row int) {
	for y := range 8 {
		copy(Screen[row*8+y][col*8:col*8+8], Back[row*8+y][col*8:col*8+8])
	}
	dirty = true
}

// FillBack fills rectangle r of the working buffer with the plane byte v (driver 0835):
// every plane gets v, so pixel x of each byte is colour 15 where bit 7-x of v is set and 0
// elsewhere.
func FillBack(r Rect, v uint8) {
	for y := r.Y * 8; y < (r.Y+r.H)*8 && y < H; y++ {
		for x := r.X * 8; x < (r.X+r.W)*8 && x < W; x++ {
			c := uint8(0)
			if v&(0x80>>(x&7)) != 0 {
				c = 15
			}
			Back[y][x] = c
		}
	}
}

// ShowWindow copies rectangle r from the working buffer to the screen (driver 08b3). It
// ignores which window owns the cells.
func ShowWindow(r Rect) {
	for y := r.Y * 8; y < (r.Y+r.H)*8 && y < H; y++ {
		x0, x1 := r.X*8, min((r.X+r.W)*8, W)
		copy(Screen[y][x0:x1], Back[y][x0:x1])
	}
	dirty = true
	Present()
}

// maze is the MAZEDATA tile set (driver 0731).
var maze *dbs.MazeData

// LoadMaze takes the MAZEDATA tiles (driver 0731, drv_load_from_file).
func LoadMaze(m *dbs.MazeData) { maze = m }

// MazeTile draws maze wall tile draw into the working buffer (driver 0b93, drv_op_c6).
// TODO: not ported yet; WMAZE's view needs it.
func MazeTile(draw int) { _ = maze }
