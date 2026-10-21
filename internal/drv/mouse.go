package drv

import "github.com/retro-ware/wiz6-go/internal/host"

var (
	mouseOn bool // MouseInit ran and MouseShutdown has not
	latched bool // a button went down since the last ClearClick or reported click
)

// MouseInit resets the mouse and shows the pointer (driver 0fae).
func MouseInit() {
	mouseOn, latched = true, false
	dirty = true
	Present()
}

// MouseShutdown hides the pointer (driver 100a).
func MouseShutdown() {
	mouseOn = false
	dirty = true
	Present()
}

// ClearClick forgets a latched click (driver 1a2e).
func ClearClick() {
	pump()
	latched = false
}

// pump takes the host's mouse events: a button going down latches a click.
func pump() {
	for {
		e, ok := host.ReadMouse()
		if !ok {
			return
		}
		if e.Button >= 0 && e.Down {
			latched = true
		}
	}
}

// pointerCell is the text cell under the pointer.
func pointerCell() (col, row int, ok bool) {
	x, y, ok := host.Pointer()
	if !ok || x < 0 || y < 0 || x >= W || y >= H {
		return 0, 0, false
	}
	return int(x) / 8, int(y) / 8, true
}

// PollClick finds the region of regions (4 bytes each: x1, y1, x2, y2 in cells, inclusive;
// a first byte of 255 ends the table) under the pointer (driver 10f7). It returns its 1-based
// number, negated when a click is latched (which it then forgets), or 0 for none.
func PollClick(regions []byte) int {
	pump()
	dirty = true
	Present() // the pointer moves
	col, row, ok := pointerCell()
	if !ok {
		return 0
	}
	for i := 0; i+3 < len(regions) && regions[i] != 255; i += 4 {
		if col >= int(regions[i]) && row >= int(regions[i+1]) && col <= int(regions[i+2]) && row <= int(regions[i+3]) {
			n := i/4 + 1
			if latched {
				latched = false
				return -n
			}
			return n
		}
	}
	return 0
}

// arrow is the pointer: 1 white, 2 black outline.
// ponytail: a plain arrow; the driver's own pointer sprite is built at run time and not decoded yet.
var arrow = [10]string{
	"2.........",
	"22........",
	"212.......",
	"2112......",
	"21112.....",
	"211112....",
	"2111112...",
	"2112222...",
	"222.......",
	"2.........",
}

// drawPointer puts the pointer into a frame being presented.
func drawPointer(f []byte) {
	if !mouseOn {
		return
	}
	x, y, ok := host.Pointer()
	if !ok {
		return
	}
	for dy, s := range arrow {
		for dx, c := range s {
			px, py := int(x)+dx, int(y)+dy
			if c == '.' || px < 0 || py < 0 || px >= W || py >= H {
				continue
			}
			v := byte(1)
			if c == '2' {
				v = 0
			}
			f[py*W+px] = v
		}
	}
}
