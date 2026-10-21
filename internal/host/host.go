// Package host is the desktop around the game: a window that shows the game's 8-bit frames
// through its VGA palette, the keyboard as PC scan codes, the mouse, and a clock. The game
// runs on its own goroutine (Run) and talks to the host only through this package's calls,
// which are safe to make from that goroutine.
package host

import (
	"image"
	"image/png"
	"os"
	"sync"
	"time"

	"gpu"
)

// The largest mode the game sets (VESA 640x480); mode 13h is 320x200.
const (
	MaxWidth  = 640
	MaxHeight = 480
)

// The frame before the game presents one is mode 13h's.
const defaultWidth, defaultHeight = 320, 200

// The window opens at three times mode 13h and reads input windowTPS times a second.
const (
	windowStartWidth, windowStartHeight = 960, 600
	windowTPS                           = 240
)

const (
	paletteSize = 256
	dacBits     = 6 // bits of a DAC channel
	opaque      = 255
	// keyBreakBit is set in a break (key up) scan code.
	keyBreakBit = 0x80
)

// A Color is a VGA DAC entry: 6 bits each of red, green and blue.
type Color struct{ R, G, B uint8 }

var state struct {
	mu                      sync.Mutex
	frameWidth, frameHeight int                        // the presented frame's size
	frame                   [MaxWidth * MaxHeight]byte // the presented frame, palette indices
	palette                 [paletteSize]Color
	rgbaFrame               []byte
	keys                    []byte       // scan codes not yet read: make codes, and break codes with bit 7 set
	mouse                   []MouseEvent // mouse events not yet read
	quit                    bool
	start                   time.Time
	screenshotPath          string // a PNG to write of the next frame shown

	pointerX, pointerY float64       // the pointer on the frame, in its pixels
	pointerSet         bool          // the pointer has a position (the window or a script set it)
	pointerWarped      bool          // the game moved the pointer: the window moves the OS pointer
	lockUntil          time.Duration // the pointer stays locked until then (RequestPointerLock)
	locked             bool          // the window has the pointer locked
	frameSeq           uint64        // frames the window has shown (pace.go)
	changed            time.Duration // when the frame or palette last changed (Now)
}

func init() {
	state.frameWidth, state.frameHeight, state.start = defaultWidth, defaultHeight, time.Now()
}

// Scripted makes the window ignore the real keyboard and mouse: only PushKey, MouseMove,
// MouseTo and MouseButton reach the game, so a scripted run can't pick up the user's
// keystrokes.
var Scripted bool

// Present shows a frame of w x h palette indices.
func Present(px []byte, w, h int) {
	state.mu.Lock()
	if w != state.frameWidth || h != state.frameHeight || string(state.frame[:w*h]) != string(px[:w*h]) {
		state.changed = Now()
	}
	state.frameWidth, state.frameHeight = w, h
	copy(state.frame[:w*h], px)
	state.mu.Unlock()
}

// SetPalette sets DAC entries first.. from cs.
func SetPalette(first int, cs []Color) {
	state.mu.Lock()
	for i, c := range cs {
		if state.palette[first+i] != c {
			state.changed = Now()
		}
	}
	copy(state.palette[first:], cs)
	state.mu.Unlock()
}

// Unchanged is how long the frame and palette have stayed the same.
func Unchanged() time.Duration {
	state.mu.Lock()
	defer state.mu.Unlock()
	return Now() - state.changed
}

// Palette returns the DAC.
func Palette() [paletteSize]Color {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.palette
}

// ReadKey returns the next scan code the keyboard sent, or false.
func ReadKey() (byte, bool) {
	state.mu.Lock()
	defer state.mu.Unlock()
	if len(state.keys) == 0 {
		return 0, false
	}
	k := state.keys[0]
	state.keys = state.keys[1:]
	return k, true
}

// PushKey queues a scan code, as a script or the window sends one.
func PushKey(code byte) {
	state.mu.Lock()
	state.keys = append(state.keys, code)
	state.mu.Unlock()
}

// A MouseEvent is relative mouse motion (Button < 0: DX, DY in counts, the OS's raw motion
// while the pointer is locked) or a button (0 left, 1 right, 2 middle) going down or up.
type MouseEvent struct {
	DX, DY float64
	Button int
	Down   bool
}

// ReadMouse returns the next mouse event, or false.
func ReadMouse() (MouseEvent, bool) {
	state.mu.Lock()
	defer state.mu.Unlock()
	if len(state.mouse) == 0 {
		return MouseEvent{}, false
	}
	e := state.mouse[0]
	state.mouse = state.mouse[1:]
	return e, true
}

// MouseTo puts the pointer at (x, y) on the frame, as a script or the window does.
func MouseTo(x, y float64) {
	state.mu.Lock()
	state.pointerX, state.pointerY, state.pointerSet = x, y, true
	state.mu.Unlock()
}

// MouseMove queues relative motion, as a script or the window (with the pointer locked)
// sends it.
func MouseMove(dx, dy float64) {
	state.mu.Lock()
	state.mouse = append(state.mouse, MouseEvent{DX: dx, DY: dy, Button: -1})
	state.mu.Unlock()
}

// MouseButton queues a button going down or up.
func MouseButton(b int, down bool) {
	state.mu.Lock()
	state.mouse = append(state.mouse, MouseEvent{Button: b, Down: down})
	state.mu.Unlock()
}

// Pointer is the pointer's position on the frame in its pixels; false before it has one.
func Pointer() (x, y float64, ok bool) {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.pointerX, state.pointerY, state.pointerSet
}

// SetPointer moves the pointer to (x, y) on the frame (the game placing its cursor).
func SetPointer(x, y float64) {
	state.mu.Lock()
	state.pointerX, state.pointerY, state.pointerSet, state.pointerWarped = x, y, true, true
	state.mu.Unlock()
}

// pointerLockHold is how long a lock request lasts: the game asks each frame it wants
// relative motion, so a screen that stops asking gets the pointer back by itself.
const pointerLockHold = 150 * time.Millisecond

// RequestPointerLock asks the window to lock the pointer (hidden, raw relative motion) for a
// little while.
func RequestPointerLock() {
	state.mu.Lock()
	state.lockUntil = Now() + pointerLockHold
	state.mu.Unlock()
}

// PointerLocked is whether the window has the pointer locked.
func PointerLocked() bool {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.locked
}

// Quit asks the window to close.
func Quit() {
	state.mu.Lock()
	state.quit = true
	state.mu.Unlock()
}

// Now is the time since the program started; a deterministic run replaces it.
var Now = func() time.Duration { return time.Since(state.start) }

// Sleep waits a little: the game's busy waits call it so they don't spin.
var Sleep = func() { time.Sleep(time.Millisecond) }

// Screenshot writes the next frame shown to a PNG.
func Screenshot(path string) {
	state.mu.Lock()
	state.screenshotPath = path
	state.mu.Unlock()
}

// FrameRGBA returns the presented frame as RGBA, with its size.
func FrameRGBA() ([]byte, int, int) {
	state.mu.Lock()
	defer state.mu.Unlock()
	return append([]byte(nil), rgbaFrameLocked()...), state.frameWidth, state.frameHeight
}

func rgbaFrameLocked() []byte {
	n := state.frameWidth * state.frameHeight
	if len(state.rgbaFrame) != n*4 {
		state.rgbaFrame = make([]byte, n*4)
	}
	var lut [paletteSize][4]byte
	for i, c := range state.palette {
		lut[i] = [4]byte{dacTo8Bits(c.R), dacTo8Bits(c.G), dacTo8Bits(c.B), opaque}
	}
	for i, p := range state.frame[:n] {
		copy(state.rgbaFrame[i*4:], lut[p][:])
	}
	return state.rgbaFrame
}

// dacTo8Bits widens a 6-bit DAC channel to 8 bits, repeating its top bits below.
func dacTo8Bits(v uint8) uint8 { return v<<(8-dacBits) | v>>(2*dacBits-8) }

// WritePNG writes RGBA pixels to a PNG file.
func WritePNG(path string, px []byte, w, h int) error {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	copy(img.Pix, px)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// Run opens the window and runs game on its own goroutine until it returns or the window
// closes. The window reads input at 240 Hz, so a game frame always sees fresh input.
func Run(title string, game func()) error {
	done := make(chan struct{})
	go func() {
		defer close(done)
		game()
	}()
	gpu.SetWindowTitle(title)
	gpu.SetWindowSize(windowStartWidth, windowStartHeight)
	gpu.SetWindowResizingMode(gpu.WindowResizingModeEnabled)
	gpu.SetTPS(windowTPS)
	gpu.SetRunnableOnUnfocused(true)
	return gpu.RunGame(&window{done: done})
}

// Headless runs game without a window (a scripted run, or a test).
func Headless(game func()) { game() }
