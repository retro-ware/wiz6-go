package drv

import (
	"strconv"

	"github.com/retro-ware/wiz6-go/internal/host"
)

var (
	pointerShown bool // MouseInit ran and MouseShutdown has not
	clickLatched bool // a button went down since the last ClearClick or reported click
)

// MouseInit resets the mouse and shows the pointer (driver 0fae).
func MouseInit() {
	pointerShown, clickLatched = true, false
	screenChanged = true
	Present()
}

// MouseShutdown hides the pointer (driver 100a).
func MouseShutdown() {
	pointerShown = false
	screenChanged = true
	Present()
}

// ClearClick forgets a latched click (driver 1a2e).
func ClearClick() {
	readMouseEvents()
	clickLatched = false
}

// readMouseEvents takes the host's mouse events: a button going down latches a click.
func readMouseEvents() {
	for {
		e, ok := host.ReadMouse()
		if !ok {
			return
		}
		if e.Button >= 0 && e.Down {
			clickLatched = true
		}
	}
}

// The PollClick region table: per region x1, y1, x2, y2 in cells; a first byte of
// regionTableEnd ends it.
const (
	regionSize     = 4
	regionTableEnd = 255
)

// pointerCell is the text cell under the pointer.
func pointerCell() (col, row int, ok bool) {
	x, y, ok := host.Pointer()
	if !ok || x < 0 || y < 0 || x >= ScreenWidth || y >= ScreenHeight {
		return 0, 0, false
	}
	return int(x) / CellSize, int(y) / CellSize, true
}

// PollClick finds the region of regions (4 bytes each: x1, y1, x2, y2 in cells, inclusive;
// a first byte of 255 ends the table) under the pointer (driver 10f7). It returns its 1-based
// number, negated when a click is latched (which it then forgets), or 0 for none.
func PollClick(regions []byte) int {
	readMouseEvents()
	screenChanged = true
	Present() // the pointer moves
	col, row, ok := pointerCell()
	if !ok {
		return 0
	}
	for i := 0; i+regionSize <= len(regions) && regions[i] != regionTableEnd; i += regionSize {
		r := regions[i : i+regionSize]
		left, top, right, bottom := int(r[0]), int(r[1]), int(r[2]), int(r[3])
		if col >= left && row >= top && col <= right && row <= bottom {
			n := i/regionSize + 1
			if clickLatched {
				clickLatched = false
				return -n
			}
			return n
		}
	}
	return 0
}

// pointerImage is the pointer as the original shows it (decoded from DOSBox-X captures): hex colour
// indices (drv.Palette), '.' transparent; the hotspot is the top-left pixel.
var pointerImage = [12]string{
	"000000000...",
	"055555550...",
	"054ccc5d0...",
	"0514c5d0....",
	"05445500....",
	"054555d00...",
	"055d055d00..",
	"05d00055d00.",
	"000..0055d00",
	"......0055d0",
	".......00500",
	"........000.",
}

// drawPointer puts the pointer into a frame being presented.
func drawPointer(f []byte) {
	if !pointerShown {
		return
	}
	x, y, ok := host.Pointer()
	if !ok {
		return
	}
	for dy, s := range pointerImage {
		for dx, c := range s {
			px, py := int(x)+dx, int(y)+dy
			if c == '.' || px < 0 || py < 0 || px >= ScreenWidth || py >= ScreenHeight {
				continue
			}
			v, _ := strconv.ParseUint(string(c), 16, 8) // a hex digit
			f[py*ScreenWidth+px] = byte(v)
		}
	}
}

// ClickPending is whether a click is latched (it stays latched for PollClick).
func ClickPending() bool {
	readMouseEvents()
	return clickLatched
}
