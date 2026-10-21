package gpu

import "github.com/gogpu/gpucontext"

var capture struct {
	captured       bool
	deltaX, deltaY float64 // locked motion since the last CursorDelta
}

// SetCursorCaptured hides and locks the pointer to the window (for relative
// mouse motion) or releases it.
func SetCursorCaptured(on bool) {
	if win.app == nil || capture.captured == on {
		return
	}
	capture.captured = on
	if on {
		win.app.SetCursorMode(gpucontext.CursorModeLocked)
	} else {
		win.app.SetCursorMode(gpucontext.CursorModeNormal)
	}
	capture.deltaX, capture.deltaY = 0, 0
}

// CursorCaptured reports whether the pointer is captured.
func CursorCaptured() bool { return capture.captured }

// CursorDelta is the pointer motion in screen points since the last call.
func CursorDelta() (float64, float64) {
	dx, dy := capture.deltaX, capture.deltaY
	capture.deltaX, capture.deltaY = 0, 0
	return dx, dy
}

// IsFocused reports whether the window has the input focus.
func IsFocused() bool { return win.app != nil && win.app.HasFocus() }

// SetCursorHidden hides the pointer while it is over the window (when not captured), for a
// game that draws its own cursor.
func SetCursorHidden(on bool) {
	if win.app == nil || capture.captured {
		return
	}
	shape := gpucontext.CursorDefault
	if on {
		shape = gpucontext.CursorNone
	}
	win.app.SetCursor(shape)
}
