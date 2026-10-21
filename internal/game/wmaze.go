package game

import "github.com/retro-ware/wiz6-go/internal/drv"

// WMAZE: exploration, the maze view and the cell events (game states 5, 6, 23; main
// turns 25 into 5).

// ovmain (WMAZE 4572): flee (23), load the map game_state_arg (6), explore (5).
func wmazeOvmain() {
	if gameState == 23 {
		fleeParty(random(10) + 4)
	}
	if gameState == 6 {
		loadMazeLevel(gameStateArg)
	}
	if gameState == 5 {
		mazeMainLoop()
	}
}

// WMAZE data segment (bss 0x501e..0x5240). The view's own tables (0x5040..0x5228) live
// in wmaze_view.go.
var (
	mzMsg2000, mzMsg2001 string // g501e, g5028: messages 2000 and 2001 (get_maze_input prompts)

	mzRedrawView  int16 // g5032: redraw the view at the top of the main loop
	mzG5034       int16 // g5034
	mzAnimCounter int16 // g5036: counts to 10 between animation frames
	mzG5038       int16 // g5038
	mzG503a       int16 // g503a: message id (2020, 2053)
	mzG503c       int16 // g503c
	mzG503e       int16 // g503e

	textWin          *Window  // g522a: the event text window (rows 19..23)
	mzG522c          int16    // g522c: lines on the text page
	mzG522e          int16    // g522e
	eventScreenDirty int16    // event_screen_dirty
	mzG5232          int16    // g5232
	lockTumblers     [6]int16 // lock_tumblers (12 bytes)
)

// drv entry points as the root calls them.
func drvMazeTile(tile, flag, mirror int) { drv.MazeTile(tile, flag, mirror) }   // drv_op_c6
func drvShowWindow(w *Window)            { drv.ShowWindow(winRect(w)) }         // drv_op_c2
func drvFillBack(w *Window, v int)       { drv.FillBack(winRect(w), uint8(v)) } // drv_op_be
