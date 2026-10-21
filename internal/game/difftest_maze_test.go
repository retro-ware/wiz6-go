package game

// mazeFuncs: WMAZE and WDOPT functions.
var mazeFuncs = map[string]func(a []int) int{
	"f55b7": func(a []int) int { mzWins(); tickEffectTimers(a[0]); return 0 },
	"f61f8": func(a []int) int { mzWins(); healMember(a[0], a[1], a[2], a[3], a[4]); return 0 },
	"f63f4": func(a []int) int { mzWins(); return b2i(restParty()) },
	"fae49": func(a []int) int { mzWins(); trapEvent(); return 0 },
	"f5949": func(a []int) int { mzWins(); moveParty(a[0]); return 0 },
	"f56f4": func(a []int) int { mzWins(); return tryMove(a[0]) },
	"f7868": func(a []int) int { mzWins(); fleeParty(a[0]); return 0 },
	"fc8eb": func(a []int) int { mzWins(); runCellEvents(); return 0 },
	"fa4f5": func(a []int) int { mzWins(); checkSecretDetect(); return 0 },
	"f6054": func(a []int) int { mzWins(); applySpellOutsideCombat(a[0], a[1], a[2], a[3]); return 0 },
	"f517b": func(a []int) int { mzWins(); return tryOpenLock(a[0]) },
	"f6561": func(a []int) int { mzWins(); return b2i(wdoptItemLocationEvent(a[0], a[1])) },
	"f4faf": func(a []int) int { mzWins(); useItemCharge(a[0], a[1]); return 0 },
	"f4e09": func(a []int) int { mzWins(); removeItem(a[0], a[1]); return 0 },
	"fdb1e": func(a []int) int { mzWins(); mzPrompts(); handleLockedDoor(); return 0 },
	"f4735": func(a []int) int { mzWins(); floodMarkCells(a[0], a[1], a[2], a[3]); return 0 },
	"f9a24": func(a []int) int { mzWins(); trySkillGain(a[0], a[1], a[2]); return 0 },
	"fced8": func(a []int) int { mzWins(); return forceLock(a[0], a[1]) },
	"fd3b3": func(a []int) int { mzWins(); return pickLock(a[0], a[1]) },
	"fd8a9": func(a []int) int { mzWins(); mzPrompts(); lockedDoorMenu(a[0]); return 0 },
	"fbc93": func(a []int) int { mzWins(); mzPrompts(); puzzleEvent(); return 0 },
	"fc182": func(a []int) int { mzWins(); mzPrompts(); riddleEvent(); return 0 },
	"f7f30": func(a []int) int { mzWins(); castSpellInMaze(a[0]); return 0 },
	"f7860": func(a []int) int { mzWins(); useItemInMaze(a[0]); return 0 },
	"f467b": func(a []int) int { mzWins(); return randomEncounterMonster(a[0]) },
	"f46e8": func(a []int) int { mzWins(); return encounterTable(a[0]) },
	"fccd2": func(a []int) int { mzWins(); searchCell(); return 0 },
}

// mzWins makes fresh windows for the maze code to draw in (the C's are stubbed); the
// windows of earlier cases go.
func mzWins() {
	windows, cellOwner = nil, [len(cellOwner)]uint8{}
	partyPictureWin = createWindow(0, 0, 40, 5, 8, -1, 0, false)
	for _, w := range []**Window{&mainWindow, &bottomLineWin, &pictureWin, &g4fac, &g4fb0, &g4fb8, &g4fba, &textWin} {
		*w = createWindow(0, 0, 40, 25, 8, -1, 0, false)
	}
}

// mzPrompts sets init_maze_screen's key sets (the cases set the C's).
func mzPrompts() {
	mzMsg2000, mzMsg2001 = string(msgs.Get(2000)), string(msgs.Get(2001))
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
	"returnCmd": &returnCmd, "g363e": &g363e, "reachableCells": &reachableCells, "eventCells": &eventCells,
	"g4338": &g4338, "g433a": &g433a, "treasureID": &treasureID,
	"lockTumblers": &lockTumblers, "msgNotFound": &msgNotFound, "dpTarget": &dpTarget, "dpResultMsg": &dpResultMsg, "dpResultBeep": &dpResultBeep,
}

func init() {
	for k, v := range mazeFuncs {
		diffFuncs[k] = v
	}
	for k, v := range mazeRoots {
		diffRoots[k] = v
	}
}
