// Package drv reimplements EGA.DRV, the original's graphics driver, on a 320x200 frame of
// 16 EGA colours. It keeps what the driver keeps (the screen, the working picture buffer,
// the pristine copy of the last full-screen picture, picture slots, fonts and the maze
// tiles) as typed Go values and does what each driver entry point does to them. There are
// no planes, ports or segments: a pixel is a colour index 0..15.
//
// Entry points, by their far-pointer slot in the root (cs:1b8a + 4*n) and driver address:
//
//	1b8a 029a  Glyph       mono glyph of font 0 in a text cell (put_char, refresh_window)
//	1b8e 053a  ColorGlyph  colour glyph of font 1..4 in a text cell (put_char_hi, borders)
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
//	1bbe 0835  FillWorkBuffer a window's rectangle of the working buffer (drv_op_be)
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
	ScreenWidth, ScreenHeight = 320, 200
	Cols, Rows                = 40, 25
	// CellSize is a text cell's width and height in pixels; a byte of a plane holds one
	// cell row.
	CellSize = 8
)

const (
	colorCount     = 16  // EGA colours
	glyphCount     = 128 // glyphs per font; a character code's bit 7 is ignored
	colorFontCount = 5   // fonts 0..4; colour fonts use 1..4
	fillCellColor  = 8   // dark grey, the colour FillCell paints
	planeCount     = 4   // EGA bit planes; plane p is bit p of a colour
	// dacShift turns an 8-bit colour channel into the VGA DAC's 6 bits.
	dacShift = 2
)

// Transparent is the colour a picture pixel with all four planes set has: draw_text and
// the maze blits leave the pixel below.
const Transparent = 15

// A Rect is a text-cell rectangle (a window's x, y, width, height).
type Rect struct{ X, Y, W, H int }

var (
	// Screen is the EGA screen (segment A000).
	Screen [ScreenHeight][ScreenWidth]uint8
	// WorkBuffer is the working picture buffer (cs:[0x14d]): picture windows show it, draw_text
	// and the maze draw into it.
	WorkBuffer     [ScreenHeight][ScreenWidth]uint8
	monoFont       [glyphCount][CellSize]uint8                           // font 0: one bit per pixel, bit 7 leftmost
	colorFonts     [colorFontCount][glyphCount][CellSize][CellSize]uint8 // fonts 1..4: colour indices
	screenChanged  bool
	presentedFrame [ScreenWidth * ScreenHeight]byte
)

// Palette is the EGA palette the root sets for mode 0Dh (int 10h ax=1002h, root 1e43),
// as RGB with 8 bits each. Colour 15 is cyan; in pictures it means transparent.
var Palette = [colorCount][3]uint8{
	{0, 0, 0}, {255, 255, 255}, {85, 85, 255}, {255, 85, 255},
	{255, 85, 85}, {255, 255, 85}, {85, 255, 85}, {85, 255, 255},
	{85, 85, 85}, {170, 170, 170}, {0, 0, 170}, {170, 0, 170},
	{170, 0, 0}, {170, 85, 0}, {0, 170, 0}, {0, 170, 170},
}

// Init sets the host palette and clears the screen, as the root's mode set does.
func Init() {
	cs := make([]host.Color, colorCount)
	for i, c := range Palette {
		cs[i] = host.Color{R: c[0] >> dacShift, G: c[1] >> dacShift, B: c[2] >> dacShift}
	}
	host.SetPalette(0, cs)
	Screen = [ScreenHeight][ScreenWidth]uint8{}
	screenChanged = true
	Present()
}

// SetMonoFont loads font 0 (WFONT0): row bytes, bit 7 the leftmost pixel.
func SetMonoFont(g [glyphCount][CellSize]uint8) { monoFont = g }

// SetFont loads colour font n (1..4): 8x8 colour indices per glyph.
func SetFont(n int, g [glyphCount][CellSize][CellSize]uint8) { colorFonts[n] = g }

// SetGlyphs replaces count glyphs of colour font n from first on (root f2256).
func SetGlyphs(n, first int, g [][CellSize][CellSize]uint8) { copy(colorFonts[n][first:], g) }

// Present hands the screen to the host if it changed since the last call.
func Present() {
	if !screenChanged {
		return
	}
	screenChanged = false
	for y := range Screen {
		copy(presentedFrame[y*ScreenWidth:], Screen[y][:])
	}
	drawPointer(presentedFrame[:])
	host.Present(presentedFrame[:], ScreenWidth, ScreenHeight)
}

// Glyph draws glyph ch of font 0 in text cell (col, row) in colour fg (driver 029a). With
// inverse the glyph's clear pixels take fg and its set pixels colour 0.
func Glyph(col, row int, ch uint8, fg uint8, inverse bool) {
	g := monoFont[ch%glyphCount]
	for y := range CellSize {
		for x := range CellSize {
			on := g[y]&pixelBit(x) != 0
			c := uint8(0)
			if on != inverse {
				c = fg % colorCount
			}
			Screen[row*CellSize+y][col*CellSize+x] = c
		}
	}
	screenChanged = true
}

// ColorGlyph draws glyph ch of colour font n (1..4) in text cell (col, row) (driver 053a).
func ColorGlyph(col, row int, ch uint8, n int) {
	g := &colorFonts[n][ch%glyphCount]
	for y := range CellSize {
		copy(Screen[row*CellSize+y][col*CellSize:(col+1)*CellSize], g[y][:])
	}
	screenChanged = true
}

// FillCell paints text cell (col, row) in colour 8 (driver 065f, write mode 2).
func FillCell(col, row int) {
	for y := range CellSize {
		for x := range CellSize {
			Screen[row*CellSize+y][col*CellSize+x] = fillCellColor
		}
	}
	screenChanged = true
}

// CopyCell copies text cell (col, row) from the working buffer to the screen (driver 0e2f).
func CopyCell(col, row int) {
	for y := range CellSize {
		x0, x1 := col*CellSize, (col+1)*CellSize
		copy(Screen[row*CellSize+y][x0:x1], WorkBuffer[row*CellSize+y][x0:x1])
	}
	screenChanged = true
}

// FillWorkBuffer fills rectangle r of the working buffer with the plane byte v (driver 0835):
// every plane gets v, so pixel x of each byte is colour 15 where bit 7-x of v is set and 0
// elsewhere.
func FillWorkBuffer(r Rect, v uint8) {
	for y := r.Y * CellSize; y < (r.Y+r.H)*CellSize && y < ScreenHeight; y++ {
		for x := r.X * CellSize; x < (r.X+r.W)*CellSize && x < ScreenWidth; x++ {
			c := uint8(0)
			if v&pixelBit(x%CellSize) != 0 {
				c = Transparent
			}
			WorkBuffer[y][x] = c
		}
	}
}

// ShowWindow copies rectangle r from the working buffer to the screen (driver 08b3). It
// ignores which window owns the cells.
func ShowWindow(r Rect) {
	for y := r.Y * CellSize; y < (r.Y+r.H)*CellSize && y < ScreenHeight; y++ {
		x0, x1 := r.X*CellSize, min((r.X+r.W)*CellSize, ScreenWidth)
		copy(Screen[y][x0:x1], WorkBuffer[y][x0:x1])
	}
	screenChanged = true
	Present()
}

// mazeData is the MAZEDATA tile set (driver 0731).
var mazeData *dbs.MazeData

// LoadMaze takes the MAZEDATA tiles (driver 0731, drv_load_from_file).
func LoadMaze(m *dbs.MazeData) { mazeData = m }

// pixelBit is the bit of a plane byte (or a font 0 row) that holds pixel x (0..7) of its
// cell row: bit 7 is the leftmost pixel.
func pixelBit(x int) byte { return 1 << (CellSize - 1 - x) }
