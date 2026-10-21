package gpu

import (
	"github.com/gogpu/gogpu"
	"github.com/gogpu/gpucontext"
)

// Key is a keyboard key: gogpu's key codes, plus the either-side modifiers.
type Key int

const (
	KeySpace     = Key(gpucontext.KeySpace)
	KeyComma     = Key(gpucontext.KeyComma)
	KeyMinus     = Key(gpucontext.KeyMinus)
	KeyPeriod    = Key(gpucontext.KeyPeriod)
	KeySlash     = Key(gpucontext.KeySlash)
	Key0         = Key(gpucontext.Key0)
	Key1         = Key0 + 1
	Key2         = Key0 + 2
	Key3         = Key0 + 3
	Key4         = Key0 + 4
	Key5         = Key0 + 5
	Key6         = Key0 + 6
	Key7         = Key0 + 7
	Key8         = Key0 + 8
	Key9         = Key0 + 9
	KeyEqual     = Key(gpucontext.KeyEqual)
	KeyA         = Key(gpucontext.KeyA)
	KeyB         = KeyA + 1
	KeyC         = KeyA + 2
	KeyD         = KeyA + 3
	KeyE         = KeyA + 4
	KeyF         = KeyA + 5
	KeyG         = KeyA + 6
	KeyH         = KeyA + 7
	KeyI         = KeyA + 8
	KeyJ         = KeyA + 9
	KeyK         = KeyA + 10
	KeyL         = KeyA + 11
	KeyM         = KeyA + 12
	KeyN         = KeyA + 13
	KeyO         = KeyA + 14
	KeyP         = KeyA + 15
	KeyQ         = KeyA + 16
	KeyR         = KeyA + 17
	KeyS         = KeyA + 18
	KeyT         = KeyA + 19
	KeyU         = KeyA + 20
	KeyV         = KeyA + 21
	KeyW         = KeyA + 22
	KeyX         = KeyA + 23
	KeyY         = KeyA + 24
	KeyZ         = KeyA + 25
	KeyEscape    = Key(gpucontext.KeyEscape)
	KeyEnter     = Key(gpucontext.KeyEnter)
	KeyTab       = Key(gpucontext.KeyTab)
	KeyBackspace = Key(gpucontext.KeyBackspace)
	KeyInsert    = Key(gpucontext.KeyInsert)
	KeyDelete    = Key(gpucontext.KeyDelete)
	KeyRight     = Key(gpucontext.KeyRight)
	KeyLeft      = Key(gpucontext.KeyLeft)
	KeyDown      = Key(gpucontext.KeyDown)
	KeyUp        = Key(gpucontext.KeyUp)
	KeyPageUp    = Key(gpucontext.KeyPageUp)
	KeyPageDown  = Key(gpucontext.KeyPageDown)
	KeyHome      = Key(gpucontext.KeyHome)
	KeyEnd       = Key(gpucontext.KeyEnd)
	KeyF1        = Key(gpucontext.KeyF1)
	KeyF2        = KeyF1 + 1
	KeyF3        = KeyF1 + 2
	KeyF4        = KeyF1 + 3
	KeyF5        = KeyF1 + 4
	KeyF6        = KeyF1 + 5
	KeyF7        = KeyF1 + 6
	KeyF8        = KeyF1 + 7
	KeyF9        = KeyF1 + 8
	KeyF10       = KeyF1 + 9
	KeyF11       = KeyF1 + 10
	KeyF12       = KeyF1 + 11
	KeyKP0       = Key(gpucontext.KeyNumpad0)
	KeyKP1       = KeyKP0 + 1
	KeyKP2       = KeyKP0 + 2
	KeyKP3       = KeyKP0 + 3
	KeyKP4       = KeyKP0 + 4
	KeyKP5       = KeyKP0 + 5
	KeyKP6       = KeyKP0 + 6
	KeyKP7       = KeyKP0 + 7
	KeyKP8       = KeyKP0 + 8
	KeyKP9       = KeyKP0 + 9
	KeyKPEnter   = Key(gpucontext.KeyNumpadEnter)

	KeyArrowUp    = KeyUp
	KeyArrowDown  = KeyDown
	KeyArrowLeft  = KeyLeft
	KeyArrowRight = KeyRight

	keyCount = 512 // gogpu's codes stay below

	// either side
	KeyShift Key = keyCount + iota
	KeyControl
	KeyAlt
	KeyMeta
	keyAll
)

var keySides = map[Key][2]gpucontext.Key{
	KeyShift:   {gpucontext.KeyLeftShift, gpucontext.KeyRightShift},
	KeyControl: {gpucontext.KeyLeftControl, gpucontext.KeyRightControl},
	KeyAlt:     {gpucontext.KeyLeftAlt, gpucontext.KeyRightAlt},
	KeyMeta:    {gpucontext.KeyLeftSuper, gpucontext.KeyRightSuper},
}

// MouseButton is a mouse button.
type MouseButton int

const (
	MouseButtonLeft   = MouseButton(gpucontext.MouseButtonLeft)
	MouseButtonRight  = MouseButton(gpucontext.MouseButtonRight)
	MouseButtonMiddle = MouseButton(gpucontext.MouseButtonMiddle)
	mouseCount        = 8
)

// GamepadID is a connected gamepad.
type GamepadID int

// StandardGamepadButton is a button of the standard layout, named by place.
type StandardGamepadButton int

const (
	StandardGamepadButtonRightBottom StandardGamepadButton = iota // A
	StandardGamepadButtonRightRight                               // B
	StandardGamepadButtonRightLeft                                // X
	StandardGamepadButtonRightTop                                 // Y
	StandardGamepadButtonFrontTopLeft
	StandardGamepadButtonFrontTopRight
	StandardGamepadButtonFrontBottomLeft  // left trigger
	StandardGamepadButtonFrontBottomRight // right trigger
	StandardGamepadButtonCenterLeft       // back
	StandardGamepadButtonCenterRight      // start
	StandardGamepadButtonLeftStick
	StandardGamepadButtonRightStick
	StandardGamepadButtonLeftTop // d-pad
	StandardGamepadButtonLeftBottom
	StandardGamepadButtonLeftLeft
	StandardGamepadButtonLeftRight
	StandardGamepadButtonCenterCenter
	padButtons
)

// StandardGamepadAxis is an axis of the standard layout.
type StandardGamepadAxis int

const (
	StandardGamepadAxisLeftStickHorizontal StandardGamepadAxis = iota
	StandardGamepadAxisLeftStickVertical
	StandardGamepadAxisRightStickHorizontal
	StandardGamepadAxisRightStickVertical
)

// TouchID is a touch; there are none on the desktop.
type TouchID int

// input is the input as Update sees it: per key and button, the ticks it has
// been held (counted at each tick, so a press shorter than a tick still
// counts once).
var input struct {
	keyDown, keyPressed [keyAll]bool
	keyHeldTicks        [keyAll]int
	buttonDown          [mouseCount]bool
	buttonPressed       [mouseCount]bool
	buttonHeldTicks     [mouseCount]int
	chars               []rune // typed since the last tick
	tickChars           []rune
	wheel               [2]float64
	tickWheel           [2]float64
	cursor              [2]float64 // window units
	cursorOffset        [2]int     // SetCursorPosition's offset, in screen pixels
}

func initInput(app *gogpu.App) {
	ev := app.EventSource()
	key := func(k gpucontext.Key, down bool) {
		if int(k) >= keyCount {
			return
		}
		input.keyDown[k] = down
		input.keyPressed[k] = input.keyPressed[k] || down
	}
	ev.OnKeyPress(func(k gpucontext.Key, _ gpucontext.Modifiers) { key(k, true) })
	ev.OnKeyRelease(func(k gpucontext.Key, _ gpucontext.Modifiers) { key(k, false) })
	ev.OnTextInput(func(t string) { input.chars = append(input.chars, []rune(t)...) })
	button := func(b gpucontext.MouseButton, down bool, x, y float64) {
		input.cursor = [2]float64{x, y}
		if int(b) < mouseCount {
			input.buttonDown[b] = down
			input.buttonPressed[b] = input.buttonPressed[b] || down
		}
	}
	ev.OnMousePress(func(b gpucontext.MouseButton, x, y float64) { button(b, true, x, y) })
	ev.OnMouseRelease(func(b gpucontext.MouseButton, x, y float64) { button(b, false, x, y) })
	ev.OnMouseMove(func(x, y float64) { input.cursor = [2]float64{x, y} })
	if ps, ok := ev.(gpucontext.PointerEventSource); ok {
		ps.OnPointer(func(e gpucontext.PointerEvent) { capture.deltaX += e.DeltaX; capture.deltaY += e.DeltaY })
	}
	ev.OnScroll(func(dx, dy float64) { input.wheel[0] += dx; input.wheel[1] += dy })
}

// tickInput advances the held counts; called before each Update.
func tickInput() {
	for _, k := range []Key{KeyShift, KeyControl, KeyAlt, KeyMeta} {
		s := keySides[k]
		input.keyDown[k] = input.keyDown[s[0]] || input.keyDown[s[1]]
		input.keyPressed[k] = input.keyPressed[s[0]] || input.keyPressed[s[1]]
	}
	for k := range input.keyHeldTicks {
		if input.keyDown[k] || input.keyPressed[k] && input.keyHeldTicks[k] == 0 {
			input.keyHeldTicks[k]++
		} else {
			input.keyHeldTicks[k] = 0
		}
		input.keyPressed[k] = false
	}
	for b := range input.buttonHeldTicks {
		if input.buttonDown[b] || input.buttonPressed[b] && input.buttonHeldTicks[b] == 0 {
			input.buttonHeldTicks[b]++
		} else {
			input.buttonHeldTicks[b] = 0
		}
		input.buttonPressed[b] = false
	}
	input.tickChars, input.chars = append(input.tickChars[:0], input.chars...), input.chars[:0]
	input.tickWheel, input.wheel = input.wheel, [2]float64{}
}

func IsKeyPressed(k Key) bool     { return k >= 0 && k < keyAll && input.keyHeldTicks[k] > 0 }
func IsKeyJustPressed(k Key) bool { return KeyPressDuration(k) == 1 }
func KeyPressDuration(k Key) int {
	if k < 0 || k >= keyAll {
		return 0
	}
	return input.keyHeldTicks[k]
}

// AppendJustPressedKeys appends the keys pressed this tick.
func AppendJustPressedKeys(ks []Key) []Key {
	for k := range Key(keyCount) {
		if input.keyHeldTicks[k] == 1 {
			ks = append(ks, k)
		}
	}
	return ks
}

// AppendInputChars appends the characters typed since the last tick.
func AppendInputChars(rs []rune) []rune { return append(rs, input.tickChars...) }

func IsMouseButtonPressed(b MouseButton) bool {
	return b >= 0 && b < mouseCount && input.buttonHeldTicks[b] > 0
}
func IsMouseButtonJustPressed(b MouseButton) bool {
	return b >= 0 && b < mouseCount && input.buttonHeldTicks[b] == 1
}

// CursorPosition is the pointer's position on the screen image.
func CursorPosition() (int, int) {
	if win.app == nil || win.screenScale == 0 {
		return 0, 0
	}
	x, y := input.cursor[0], input.cursor[1]
	s := Monitor().DeviceScaleFactor()
	return int((x*s-win.screenX)/win.screenScale) + input.cursorOffset[0], int((y*s-win.screenY)/win.screenScale) + input.cursorOffset[1]
}

// Wheel is the wheel movement since the last tick.
func Wheel() (float64, float64) { return input.tickWheel[0], input.tickWheel[1] }

// Gamepads: gogpu reads none yet, so there are none.
// ponytail: wire them up when gogpu gets gamepad input
func AppendGamepadIDs(ids []GamepadID) []GamepadID                            { return ids }
func IsStandardGamepadLayoutAvailable(GamepadID) bool                         { return false }
func IsStandardGamepadButtonPressed(GamepadID, StandardGamepadButton) bool    { return false }
func StandardGamepadButtonPressDuration(GamepadID, StandardGamepadButton) int { return 0 }
func StandardGamepadAxisValue(GamepadID, StandardGamepadAxis) float64         { return 0 }

// AppendJustPressedTouchIDs: no touches on the desktop.
func AppendJustPressedTouchIDs(ids []TouchID) []TouchID { return ids }
func TouchPosition(TouchID) (int, int)                  { return 0, 0 }
