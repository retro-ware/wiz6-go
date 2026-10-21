package game

import (
	"fmt"
	"os"

	"github.com/retro-ware/wiz6-go/internal/drv"
)

// Window is a text window (the root's window struct): a rectangle of 8x8 cells on the
// 40x25 screen, its z-order priority, a cursor, eight border glyphs of font 1 drawn just
// outside it, and the cells it shows.
type Window struct {
	W, H     uint8    // size in cells (struct bytes 0, 1)
	X, Y     uint8    // position (2, 3)
	Priority uint8    // z-order: higher is on top (4)
	Flags    uint8    // WinDeferred, WinPicture (5)
	CurX     uint8    // cursor column (6)
	CurY     uint8    // cursor row (7)
	Border   [8]uint8 // top-left, top, top-right, left, right, bottom-left, bottom, bottom-right (8..15)
	Cells    []Cell   // W*H cells, row-major; none for a picture window (16..)
}

// Window flags.
const (
	WinDeferred = 1 // drawing is held back (defer_window_draw)
	WinPicture  = 2 // the window shows the working picture buffer, it has no cells
)

// A Cell is a character and its attribute: the high nibble is a font 0 colour, or the low
// nibble a colour font 1..4.
type Cell struct{ Ch, Attr uint8 }

// The ownership map (root cs:[1b60]): for each screen cell, the 1-based index in windows
// of the topmost window that owns it (low bits, 0 none), cellDirty when it must be drawn
// again, cellPointer when the mouse pointer covers it.
var cellOwner [drv.Cols * drv.Rows]uint8

const (
	cellDirty   = 0x80
	cellPointer = 0x40
)

// windows is the window list (root cs:1bd2), highest priority first.
var windows []*Window

// windowID is w's 1-based place in the window list, or 0.
func windowID(w *Window) uint8 {
	for i, v := range windows {
		if v == w {
			return uint8(i + 1)
		}
	}
	return 0
}

// outerRect is w's rectangle with its borders (root 3250).
func outerRect(w *Window) (x, y, width, height int) {
	x, y, width, height = int(int8(w.X)), int(int8(w.Y)), int(int8(w.W)), int(int8(w.H))
	if w.Border[1] != 0 {
		y--
		height++
	}
	if w.Border[3] != 0 {
		x--
		width++
	}
	if w.Border[6] != 0 {
		height++
	}
	if w.Border[4] != 0 {
		width++
	}
	return
}

// eachCell calls f with the map index of every cell of a rectangle. The map is one line of
// 1000 bytes, as in the C: a rectangle past the right edge wraps to the next row.
func eachCell(x, y, width, height int, f func(i int)) {
	for r := range height {
		for c := range width {
			if i := (y+r)*drv.Cols + x + c; i >= 0 && i < len(cellOwner) {
				f(i)
			}
		}
	}
}

// claim gives the cells of w's rectangle that no window, or a window below id, owns to
// w and marks them dirty (the tail of link_window and resume_window_draw).
func claim(w *Window, id uint8) {
	x, y, width, height := outerRect(w)
	eachCell(x, y, width, height, func(i int) {
		if o := cellOwner[i] & 0x7f; o == 0 || o >= id {
			cellOwner[i] = id | cellDirty
		}
	})
}

// fillUnowned gives the unowned cells of the windows from place id on (not deferred)
// to them (the tail of unlink_window and defer_window_draw).
func fillUnowned(id uint8) {
	for ; int(id) <= len(windows); id++ {
		w := windows[id-1]
		if w.Flags&WinDeferred != 0 {
			continue
		}
		x, y, width, height := outerRect(w)
		eachCell(x, y, width, height, func(i int) {
			if cellOwner[i]&0x7f == 0 {
				cellOwner[i] = id | cellDirty
			}
		})
	}
}

// linkWindow (ROOT 2f76, asm) puts w into the window list by priority and gives it the
// cells it covers.
func linkWindow(w *Window) {
	i := 0
	for ; i < len(windows); i++ {
		if w.Priority > windows[i].Priority {
			break
		}
		if windows[i] == w {
			return
		}
	}
	windows = append(windows, nil)
	copy(windows[i+1:], windows[i:])
	windows[i] = w
	id := uint8(i + 1)
	for j, o := range cellOwner {
		if o&0x7f >= id {
			cellOwner[j]++
		}
	}
	if w.Flags&WinDeferred != 0 {
		return
	}
	claim(w, id)
}

// unlinkWindow (ROOT 303e, asm) takes w out of the window list; the windows below get
// its cells.
func unlinkWindow(w *Window) {
	id := windowID(w)
	if id == 0 {
		return
	}
	windows = append(windows[:id-1], windows[id:]...)
	for j, o := range cellOwner {
		switch o := o & 0x7f; {
		case o == id:
			cellOwner[j] = cellDirty
		case o > id:
			cellOwner[j]--
		}
	}
	fillUnowned(id)
}

// raiseWindow (ROOT 3100, asm) puts w back in the list, in front of its priority.
func raiseWindow(w *Window) {
	unlinkWindow(w)
	linkWindow(w)
}

// deferWindowDraw (ROOT 3118, asm) holds back w's drawing and hands its cells to the
// windows below.
func deferWindowDraw(w *Window) {
	w.Flags |= WinDeferred
	id := windowID(w)
	if id == 0 {
		return
	}
	for j, o := range cellOwner {
		if o&0x7f == id {
			cellOwner[j] = cellDirty
		}
	}
	fillUnowned(id + 1)
}

// resumeWindowDraw (ROOT 31d1, asm) takes back w's cells; they draw at the next refresh.
func resumeWindowDraw(w *Window) {
	w.Flags &^= WinDeferred
	if id := windowID(w); id != 0 {
		claim(w, id)
	}
}

// fillWindow (ROOT 32be, asm) fills w with ch in attribute attr. It homes the cursor and
// marks w's cells dirty. The asm's 4th argument, shifted into the high nibble, is used
// only when attr is 0; every call in the game passes 0 there, so it is left out.
func fillWindow(w *Window, ch, attr int) {
	id := windowID(w)
	if id == 0 {
		return
	}
	w.CurX, w.CurY = 0, 0
	for j, o := range cellOwner {
		if o&0x7f == id {
			cellOwner[j] |= cellDirty
		}
	}
	for i := range w.Cells {
		w.Cells[i] = Cell{uint8(ch), uint8(attr)}
	}
}

// blankWindow (ROOT 329a, asm) fills w with spaces in colour 0.
func blankWindow(w *Window) { fillWindow(w, ' ', 0) }

// refreshWindow (ROOT 2b19, asm) draws w's dirty cells and border.
func refreshWindow(w *Window) {
	id := windowID(w)
	if id == 0 {
		return
	}
	mine := id | cellDirty
	take := func(i int) bool {
		if cellOwner[i]&0xbf != mine {
			return false
		}
		cellOwner[i] &= 0x3f
		return true
	}
	at := func(i int) (int, int) { return i % drv.Cols, i / drv.Cols }
	k := 0
	eachCellRaw(int(w.X), int(w.Y), int(w.W), int(w.H), func(i int) {
		c := Cell{}
		if k < len(w.Cells) {
			c = w.Cells[k]
		}
		k++
		if !take(i) {
			return
		}
		col, row := at(i)
		switch {
		case w.Flags&WinPicture != 0:
			drv.CopyCell(col, row)
		case c.Attr&0x0f == 0:
			drv.Glyph(col, row, c.Ch, c.Attr>>4, false)
		case c.Attr&0x0f <= 4:
			drv.ColorGlyph(col, row, c.Ch, int(c.Attr&0x0f))
		default:
			fmt.Fprintln(os.Stderr, "\aInvalid font in refreshwindow")
			os.Exit(255)
		}
	})
	b := &w.Border
	if b[1] == 0 && b[3] == 0 && b[4] == 0 && b[6] == 0 {
		drv.Present()
		return
	}
	border := func(i int, g uint8) {
		if take(i) {
			col, row := at(i)
			drv.ColorGlyph(col, row, g, 1)
		}
	}
	x, y, width, rows := outerRect(w)
	si := y*drv.Cols + x
	// The top row: the corners, then the top glyph from the cell after the left corner.
	if b[1] != 0 {
		rows--
		n := width
		if b[0] != 0 {
			n--
			border(si, b[0])
		}
		if b[2] != 0 {
			n--
		}
		i := si
		for range n {
			i++
			border(i, b[1])
		}
		if b[2] != 0 {
			i++
			border(i, b[2])
		}
		si += drv.Cols
	}
	// The sides.
	if b[3] != 0 || b[4] != 0 {
		if b[6] != 0 {
			rows--
		}
		for {
			if b[3] != 0 {
				border(si, b[3])
			}
			if b[4] != 0 {
				border(si+width-1, b[4])
			}
			rows--
			if rows == 0 {
				break
			}
			si += drv.Cols
		}
	}
	// The bottom row, one row on from where the sides (or the top) left off.
	if b[6] != 0 {
		si += drv.Cols
		n := width
		if b[5] != 0 {
			n--
			border(si, b[5])
		}
		if b[7] != 0 {
			n--
		}
		i := si
		for range n {
			i++
			border(i, b[6])
		}
		if b[7] != 0 {
			i++
			border(i, b[7])
		}
	}
	drv.Present()
}

// eachCellRaw is eachCell for a window's own rectangle, visiting out-of-map cells too so
// the cell index keeps step.
func eachCellRaw(x, y, width, height int, f func(i int)) {
	for r := range height {
		for c := range width {
			i := (y+r)*drv.Cols + x + c
			if i >= 0 && i < len(cellOwner) {
				f(i)
			} else {
				f(-1)
			}
		}
	}
}

// refreshAllWindows (ROOT 2a63, asm) refreshes every window, then paints the dirty cells
// no window owns.
func refreshAllWindows() {
	for _, w := range windows {
		refreshWindow(w)
	}
	for i, o := range cellOwner {
		if o&0xbf == cellDirty {
			drv.FillCell(i%drv.Cols, i/drv.Cols)
			cellOwner[i] &= 0x3f
		}
	}
	drv.Present()
}

// createWindow (ROOT 011a) makes a window of width x height cells at (x, y), links it and
// clears it. border -1 is none, -2..-5 are the game's border styles, any other value is
// the first of eight consecutive border glyphs of font 1.
func createWindow(x, y, width, height, color, border, flags int, refresh bool) *Window {
	w := &Window{W: uint8(width), H: uint8(height), X: uint8(x), Y: uint8(y), Priority: uint8(color), Flags: uint8(flags)}
	if flags&WinPicture == 0 {
		w.Cells = make([]Cell, width*height)
	}
	switch border {
	case -1:
	case -2:
		w.Border = [8]uint8{27, 27, 28, 27, 28, 27, 27, 28}
	case -3:
		w.Border = [8]uint8{26, 27, 27, 26, 27, 26, 27, 27}
	case -4:
		w.Border = [8]uint8{0, 0, 28, 0, 28, 0, 0, 28}
	case -5:
		w.Border = [8]uint8{35, 35, 35, 0, 0, 0, 0, 0}
	default:
		for i := range w.Border {
			w.Border[i] = uint8(border + i)
		}
	}
	linkWindow(w)
	if flags&WinPicture == 0 {
		blankWindow(w)
	}
	if refresh {
		refreshAllWindows()
	}
	return w
}

// destroyWindow (ROOT 02d2) unlinks w.
func destroyWindow(w *Window, refresh bool) {
	if w == nil {
		return
	}
	unlinkWindow(w)
	if refresh {
		refreshAllWindows()
	}
}

// clearWindow (ROOT 02f0) blanks w, drawing it with redraw.
func clearWindow(w *Window, redraw bool) {
	blankWindow(w)
	if redraw {
		refreshWindow(w)
	}
}

// setCursor (ROOT 030b) puts w's cursor at (col, row), wrapped to its size.
func setCursor(w *Window, col, row int) {
	w.CurX = uint8(col % int(w.W))
	w.CurY = uint8(row % int(w.H))
}

// clearWindowArea (ROOT 033f) prints spaces over columns col1..col2 of rows row1..row2.
func clearWindowArea(w *Window, col1, row1, col2, row2 int) {
	blanks := make([]byte, col2-col1+1)
	for i := range blanks {
		blanks[i] = ' '
	}
	for ; row1 <= row2; row1++ {
		setCursor(w, col1, row1)
		printString(w, string(blanks), 0)
	}
}

// padToColumn (ROOT 0396) prints spaces until w's cursor reaches col. Colours from 16 on
// are colour font color-15; below that the spaces are font 0 colour 1.
func padToColumn(w *Window, col, color int) {
	for int(w.CurX) < col {
		if color >= 16 {
			putCharHi(w, ' ', color-15)
		} else {
			putChar(w, ' ', 1)
		}
	}
}

// windowNewline (ROOT 03e4) moves w's cursor to the start of the next row, wrapping.
func windowNewline(w *Window) {
	w.CurX = 0
	w.CurY = uint8((int(w.CurY) + 1) % int(w.H))
}

// eraseWindow (ROOT 09fb) fills w with its background pattern (glyph 28 of font 3 for the
// main window, 30 for others) and draws it.
func eraseWindow(w *Window, noDefer bool) {
	if !noDefer {
		deferWindowDraw(w)
	}
	ch := 30
	if w == mainWindow {
		ch = 28
	}
	fillWindow(w, ch, 3)
	refreshWindow(w)
	if !noDefer {
		resumeWindowDraw(w)
	}
}

// eraseCursorMark (ROOT 0a42) puts the window's prompt mark (glyph 29 of font 3 in the
// main window, 31 elsewhere) at column 19 of row 0.
func eraseCursorMark(w *Window, noDefer bool) {
	if !noDefer {
		deferWindowDraw(w)
	}
	setCursor(w, 19, 0)
	ch := byte(31)
	if w == mainWindow {
		ch = 29
	}
	putCharHi(w, ch, 3)
	if !noDefer {
		resumeWindowDraw(w)
	}
}
