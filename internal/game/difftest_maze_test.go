package game

// mazeFuncs: WMAZE and WDOPT functions.
var mazeFuncs = map[string]func(a []int) int{
	"f55b7": func(a []int) int { mzWins(); tickEffectTimers(a[0]); return 0 },
	"f61f8": func(a []int) int { mzWins(); healMember(a[0], a[1], a[2], a[3], a[4]); return 0 },
	"f63f4": func(a []int) int { mzWins(); return b2i(restParty()) },
	"fae49": func(a []int) int { mzWins(); trapEvent(); return 0 },
}

// mzWins makes the windows the maze code draws in; the C's are stubbed.
func mzWins() {
	for _, w := range []**Window{&mainWindow, &bottomLineWin, &pictureWin, &g4fac, &g4fb0, &g4fb8, &g4fba, &textWin} {
		if *w == nil {
			*w = createWindow(0, 0, 40, 25, 8, -1, 0, false)
		}
	}
}

// mazeRoots: the globals their cases name.
var mazeRoots = map[string]any{
	"party": &party, "partyCount": &partyCount, "gameState": &gameState, "gameStateArg": &gameStateArg,
	"hdr": &hdr, "messageDelay": &messageDelay, "curSound": &curSound, "curMember": &curMember,
	"cellEventIndex": &cellEventIndex, "curEvent": &curEvent, "reloadMapA": &reloadMapA, "reloadMapB": &reloadMapB,
	"levelFlags": &levelFlags, "saveState": &saveState, "partyCounters": &partyCounters,
	"g4f84": &g4f84, "g4f86": &g4f86, "g4f88": &g4f88, "g4f8a": &g4f8a, "g4fa6": &g4fa6,
	"partyFacing": &partyFacing, "partyBlock": &partyBlock, "cellX": &cellX, "cellY": &cellY,
	"partyX": &partyX, "partyY": &partyY, "mazeA": &mazeA, "mazeB": &mazeB,
	"mzRedrawView": &mzRedrawView, "mzG5034": &mzG5034, "mzG5038": &mzG5038, "mzG503a": &mzG503a,
	"mzG503c": &mzG503c, "mzG503e": &mzG503e, "eventScreenDirty": &eventScreenDirty, "mzG5232": &mzG5232,
}

func init() {
	for k, v := range mazeFuncs {
		diffFuncs[k] = v
	}
	for k, v := range mazeRoots {
		diffRoots[k] = v
	}
}
