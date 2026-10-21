package gpu

// SetCursorPosition moves the pointer to (x, y) on the screen image (CursorPosition's
// inverse). gogpu can't move the OS pointer, so CursorPosition is offset instead: right for
// a game that hides the pointer and draws its own.
func SetCursorPosition(x, y int) {
	cx, cy := CursorPosition()
	input.cursorOffset[0] += x - cx
	input.cursorOffset[1] += y - cy
}
