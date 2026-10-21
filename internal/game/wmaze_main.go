package game

import (
	"bytes"
	"strings"
)

// WMAZE: map loading, the party panels, input, moving and the maze menus.

// memberName is party member m's name (the C passes party+432*m as a string).
func memberName(m int) string {
	n := party[m].Name[:]
	if i := bytes.IndexByte(n, 0); i >= 0 {
		n = n[:i]
	}
	return string(n)
}

// alloc_maze_buffers (WMAZE 45a6) copies map gameStateArg from the cache into the working
// maps when asked and sets the party's map position from its block.
func allocMazeBuffers() {
	if mazeA == nil {
		mazeA = new(MapA)
		reloadMapA = true
	}
	if mazeB == nil {
		mazeB = new(MapB)
		reloadMapB = true
	}
	if reloadMapA {
		*mazeA = mapCacheA[gameStateArg]
	}
	if reloadMapB {
		*mazeB = mapCacheB[gameStateArg]
	}
	reloadMapA, reloadMapB = false, false
	partyX = int16(mazeA.BlockX[partyBlock]) + cellX
	partyY = int16(mazeA.BlockY[partyBlock]) + cellY
}

// f4655 (WMAZE): one idle step between animation frames, now and then an ambient sound.
func idleStep() {
	if DetRNG || random(40) > 0 { // -detrng: no ambient roll per idle step (tools/parity patches the OVRs alike)
		delayTick()
		delayTick()
		return
	}
	playSound(random(3) + 10)
}

// f467b (WMAZE): a random encounter monster from table t of the map.
func randomEncounterMonster(t int) int {
	row := random(100) % 4
	r := random(100)
	col := 0
	if r >= 74 {
		col = 1
		if r >= 92 {
			col = 2
		}
	}
	return int(encounterByte(t*12 + row*3 + col))
}

// encounterByte is byte i of the map record from Encounters on; a table past the 8 that
// Encounters holds reads on into Walls0, as the C does.
func encounterByte(i int) uint8 {
	if i < 96 {
		return mazeA.Encounters[i/12][i%12/3][i%3]
	}
	return mazeA.Walls0[i-96]
}

// f46e8 (WMAZE): the encounter table of block blk, one of its two by its chance.
func encounterTable(blk int) int {
	i := 1
	if random(100) < int(mazeA.Unk1334[blk]) {
		i = 0
	}
	return getBitfield(mazeA.Bits1322[:], blk<<1+i, 4)
}

// flood_mark_cells (WMAZE 4735) recursively sets (v 1) or clears (v 0) the reachable mark
// of the cells joined to (x, y) in block; clearing also drops the block's start points
// there.
func floodMarkCells(block, x, y, v int) {
	x = (x + 8) % 8
	y = (y + 8) % 8
	c := cellIndex(block, x, y)
	if b2i(testBit(reachableCells[:], c)) == v {
		return
	}
	setBit(reachableCells[:], c, v != 0)
	if v == 0 {
		for i := 0; i < 5; i++ {
			sx := getBitfield(mazeA.Bits1274[:], block*5+i, 3)
			sy := getBitfield(mazeA.Bits1298[:], block*5+i, 3)
			if sx == x && sy == y {
				setBitfield(mazeA.Bits1274[:], block*5+i, 3, 0)
				setBitfield(mazeA.Bits1298[:], block*5+i, 3, 0)
			}
		}
	}
	if getBitfield(mazeA.Walls0[:], c, 2) == 0 && y < 7 {
		floodMarkCells(block, x, y+1, v)
	}
	if getBitfield(mazeA.Walls1[:], c, 2) == 0 && x < 7 {
		floodMarkCells(block, x+1, y, v)
	}
	if y > 0 && getBitfield(mazeA.Walls0[:], cellIndex(block, x, y)-8, 2) == 0 {
		floodMarkCells(block, x, y-1, v)
	}
	if x > 0 && getBitfield(mazeA.Walls1[:], cellIndex(block, x, y)-1, 2) == 0 {
		floodMarkCells(block, x-1, y, v)
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// load_maze_level (WMAZE 4937) enters map m: loads it, marks the cells with events and
// the cells reachable from the start points, and goes to state 5 (7 for no map).
func loadMazeLevel(m int) {
	gameStateArg = m
	ensureDiskFor(2, gameStateArg)
	if gameStateArg < 0 {
		gameState = 7
		return
	}
	reloadMapA, reloadMapB = true, true
	allocMazeBuffers()
	levelFlags[gameStateArg] = hdr.LevelFlags[gameStateArg]
	returnCmd = 5
	g4f8a = -1
	eventCells = [96]byte{}
	cellEventIndex = int16(mazeB.FirstEvent[partyBlock])
	for blk := 0; blk < 12; blk++ {
		for e := int(mazeB.FirstEvent[blk]); e < 144 && int(mazeB.Unk1296[e]) == blk; e++ {
			if mazeB.Kind[e] > 0 {
				setBit(eventCells[:], blk<<6|int(mazeB.Y[e])<<3|int(mazeB.X[e]), true)
			}
		}
	}
	reachableCells = [96]byte{}
	for i := 0; i < 5; i++ {
		for blk := 0; blk < 12; blk++ {
			x := getBitfield(mazeA.Bits1274[:], blk*5+i, 3)
			y := getBitfield(mazeA.Bits1298[:], blk*5+i, 3)
			if x+y > 0 {
				floodMarkCells(blk, x, y, 1)
			}
		}
	}
	if gameStateArg != 8 {
		partyCounters[0].B = 0
	}
	gameState = 5
}

// draw_vital_bar_m (WMAZE 4ac7) draws a three-cell vertical gauge of cur/max at (x, y) of
// win from glyph base on (font 1).
func drawVitalBar(win *Window, x, y, cur, max, base int) {
	n := int(int32(cur)*100/int32(max)/10) + 1
	if n > 10 {
		n = 10
	}
	if cur == 0 {
		n = 0
	}
	g := [3]int{base + 9, base + 4, base}
	if n <= 3 {
		g[2] = base + n
	} else {
		g[2] = base + 3
		n -= 3
		if n <= 4 {
			g[1] = base + n + 4
		} else {
			g[1] = base + 8
			n -= 4
			g[0] = base + n + 9
		}
	}
	for i := range g {
		setCursor(win, x, y+i)
		putCharHi(win, byte(g[i]), 1)
	}
}

// draw_member_status_m (WMAZE 4ba8) draws member m's panel: name, portrait, condition and
// status glyphs, class, the equipped hands and the hit point and stamina gauges.
func drawMemberStatus(m int) {
	win := g4fb8
	if m%2 == 0 {
		win = g4fba
	}
	x, y := 0, (m/2)<<2
	pc := &party[m]
	setCursor(win, x, y)
	name := memberName(m)
	printStringHi(win, name, 3)
	for i := len(name); i < 7; i++ {
		putCharHi(win, ' ', 3)
	}
	base := int(memberPortrait[m])*9 + 72
	for r := 0; r < 3; r++ {
		setCursor(win, x, y+r+1)
		for c := 0; c < 3; c++ {
			putCharHi(win, byte(r*3+base+c), 2)
		}
	}
	g := conditionGlyph[pc.Condition]
	if g == 0 && pc.Status[2] > 0 {
		g = 1
	}
	if g < 2 && pc.Status[3] > 0 {
		g = 2
	}
	setCursor(win, x+3, y+3)
	putCharHi(win, byte(g+37), 1)
	g = -1
	for i := 0; i < 10; i++ {
		if pc.Status[i] > 0 && statusGlyph[i] > g {
			g = statusGlyph[i]
		}
	}
	if g != -1 {
		putCharHi(win, byte(g+37), 1)
	} else {
		putCharHi(win, ' ', 3)
	}
	setCursor(win, x+3, y+2)
	for i := 0; i < 2; i++ {
		putCharHi(win, byte(int(pc.Class)<<1+i+58), 1)
	}
	setCursor(win, x+3, y+1)
	for i := 0; i < 2; i++ {
		if e := pc.Equipped[i]; e != 255 && pc.Items[e].ID > 0 {
			putCharHi(win, pc.Items[e].Icon+1, 4)
		} else {
			putCharHi(win, byte(i+37), 4)
		}
	}
	drawVitalBar(win, x+5, y+1, int(pc.HP), int(pc.HPMax), 86)
	drawVitalBar(win, x+6, y+1, int(pc.Stamina), int(pc.StaminaMax), 99)
}

// init_maze_screen (WMAZE 4e89) reads the key prompts, draws the panels and resets the
// view animation.
func initMazeScreen() {
	mzMsg2000 = getMessage(2000)
	mzMsg2001 = getMessage(2001)
	for m := 0; m < partyCount; m++ {
		drawMemberStatus(m)
	}
	mzRedrawView = 1
	mzG5034 = 1
	mzAnimCounter = 0
	animPhase = 0
	g363e = 0
}

// memberCell is where member m's name sits in the choose window.
func memberCell(m int) (int, int) { return (m%2)*9 + 2, m/2 + 1 }

// draw_member_name (WMAZE 4ee7, WBASE 6be6, WTREA 5604, ...) reprints member m's name plain in
// the member picker win.
func drawMemberName(win *Window, m int) {
	x, y := memberCell(m)
	setCursor(win, x, y)
	printStringHi(win, memberName(m), 3)
}

// choose_party_member_m (WMAZE 4f2c) asks with message msg for a party member, by arrows
// or a click; with cancel an "exit" choice comes first. The member, -1 for exit (0 with
// fewer than two members).
func choosePartyMember(msg, cancel int) int {
	if partyCount < 2 {
		return 0
	}
	win := createWindow(0, 19, 19, 5, 25, -4, 0, false)
	fillWindow(win, ' ', 3)
	eraseWindow(mainWindow, false)
	eraseCursorMark(mainWindow, false)
	refreshAllWindows()
	text := getMessage(msg)
	extra := 0
	if cancel == 1 {
		extra = 5
	}
	setCursor(mainWindow, 10-(len(text)+extra+1)/2, 0)
	printStringHi(mainWindow, text, 3)
	var exitX, exitY int
	var exitHi, exitLo string
	if cancel == 1 {
		mainWindow.CurX++
		exitX, exitY = int(mainWindow.CurX), int(mainWindow.CurY)
		exitHi = getMessage(2027)
		exitLo = getMessage(2028)
		if mousePresent {
			printStringHi(mainWindow, exitHi, 3)
		} else {
			printString(mainWindow, exitLo, -5)
		}
	}
	for m := 0; m < partyCount; m++ {
		x, y := memberCell(m)
		setCursor(win, x, y)
		printStringHi(win, memberName(m), 3)
		setClickRegion(m, win.X+byte(x), win.Y+byte(y), byte(len(memberName(m))+int(win.X)+x-1), win.Y+byte(y))
	}
	setClickRegion(partyCount, 255, 0, 0, 0)
	if cancel == 1 {
		setClickRegion(partyCount, mainWindow.X+byte(exitX), mainWindow.Y+byte(exitY), byte(len(exitLo)+int(mainWindow.X)+exitX-1), mainWindow.Y+byte(exitY))
		setClickRegion(partyCount+1, 255, 0, 0, 0)
	}
	sel := 0
	if cancel == 1 {
		sel = -1
	}
	for {
		if !mousePresent {
			if sel == -1 {
				setCursor(mainWindow, exitX, exitY)
				printString(mainWindow, exitLo, -5)
			} else {
				x, y := memberCell(sel)
				setCursor(win, x, y)
				printString(win, memberName(sel), -5)
			}
		}
		k := getMazeInput(mzMsg2001)
		if menuClicked == 1 {
			sel = -1
			if k < partyCount {
				sel = k
			}
			k = 5
		} else {
			switch k {
			case 1:
				if sel != -1 && sel%2 > 0 {
					drawMemberName(win, sel)
					sel--
				}
			case 2:
				if sel != -1 {
					if sel-2 >= 0 {
						drawMemberName(win, sel)
						sel -= 2
					} else if cancel == 1 {
						drawMemberName(win, sel)
						sel = -1
					}
				}
			case 3:
				if sel != -1 && sel%2 < 1 && sel+1 < partyCount {
					drawMemberName(win, sel)
					sel++
				}
			case 4:
				if sel == -1 {
					setCursor(mainWindow, exitX, exitY)
					printStringHi(mainWindow, exitHi, 3)
					sel = 0
				} else if sel+2 < partyCount {
					drawMemberName(win, sel)
					sel += 2
				}
			}
		}
		if k == 5 {
			break
		}
	}
	eraseWindow(mainWindow, false)
	destroyWindow(win, false)
	return sel
}

// animateStep is one tick of the idle animation: every tenth flips the view's animation
// phase and redraws it, the others idle (the body shared by wait_animating_m and
// get_maze_input).
func animateStep() {
	if mzAnimCounter == 0 {
		animPace(mazeAnimPeriod)
		animPhase = (animPhase + 1) % 2
		redrawMazeView()
	} else {
		idleStep()
	}
	mzAnimCounter = (mzAnimCounter + 1) % 10
}

// wait_animating_m (WMAZE 533b) waits n*5 animation steps; a click or Enter cuts it short
// and sets g5038.
func waitAnimating(n int) {
	flushKeys()
	for n *= 5; n > 0; n-- {
		if checkScreenClick() {
			mzG5038 = 1
			return
		}
		if keyAvailable(1) && int8(getKey(1)) == 13 {
			mzG5038 = 1
			return
		}
		animateStep()
	}
}

// show_maze_text (WMAZE 53b8) shows s centred in the main window.
func showMazeText(s string) {
	eraseWindow(mainWindow, true)
	printCentered(mainWindow, s, 0, 18)
	flushKeys()
}

// show_maze_message (WMAZE 53e5) shows message msg in the main window.
func showMazeMessage(msg int) { showMazeText(getMessage(msg)) }

// kill_party (WMAZE 5404) shows message msg and kills the whole party (state 8).
func killParty(msg int) {
	redrawMazeView()
	showMazeMessage(msg)
	for m := 0; m < partyCount; m++ {
		if party[m].Condition < 3 {
			playSound(int(party[m].Sex) + 8)
		}
		party[m].Condition = 3
		party[m].HP = 0
		party[m].Stamina = 0
		drawMemberStatus(m)
		waitAnimating(2)
	}
	waitAnimating(messageDelay << 1)
	eraseWindow(mainWindow, true)
	gameState = 8
}

// get_maze_input (WMAZE 5534) animates the view until a key in keys (its index) or a
// click on a clickRegions entry (its 0-based index, menuClicked set).
func getMazeInput(keys string) int {
	menuClicked = 0
	for {
		if keyAvailable(1) {
			if i := strings.IndexByte(keys, byte(getKey(1))); i >= 0 {
				return i
			}
			continue
		}
		r := checkClickRegions(clickRegions[:])
		if clickFlag {
			menuClicked = 1
			return r - 1
		}
		animateStep()
	}
}

// tick_effect_timers (WMAZE 55b7) counts the party effects down by n; an expired one
// (other than the first) loses its value.
func tickEffectTimers(n int) {
	for i := range partyCounters {
		c := &partyCounters[i]
		c.A -= int16(n)
		if c.A <= 0 {
			c.A = 0
			if i != 0 {
				c.B = 0
			}
		}
	}
}

// step_position (WMAZE 55ef) moves the party's map position n steps towards dir.
func stepPosition(dir int, n int16) {
	mzG5034 = 1
	curEvent = [4]int16{}
	g4f86 = 0
	switch dir {
	case 0:
		partyY += n
	case 1:
		partyX += n
	case 2:
		partyY -= n
	case 3:
		partyX -= n
	}
}

// save_position (WMAZE 5651) remembers where the party is.
func savePosition() {
	s := &saveState
	s.G4f96, s.G4f94 = partyX, partyY
	s.G4f92, s.G4f90 = cellX, cellY
	s.G4f8e = partyBlock
	s.G4f98 = int16(gameStateArg)
	s.G4f8c = partyFacing
}

// f567c (WMAZE): message msg with sound snd, held n animation waits.
func showMessageHeld(msg, snd, n int) {
	showMazeMessage(msg)
	flushKeys()
	playSound(snd)
	waitAnimating(n)
	eraseWindow(mainWindow, true)
}

// restore_position (WMAZE 56a9) puts the party back where save_position left it.
func restorePosition() {
	s := &saveState
	partyX, partyY = s.G4f96, s.G4f94
	cellX, cellY = s.G4f92, s.G4f90
	partyBlock = s.G4f8e
	partyFacing = s.G4f8c
	moveToMazeCell(&cellX, &cellY, &partyBlock, int(partyX), int(partyY))
	mzG503a = 2053
	mzG5034 = 0
}

// try_move (WMAZE 56f4) steps the party towards dir and handles falling off the map and
// pits; 1 if the party died.
func tryMove(dir int) int {
	savePosition()
	c := cellIndex(int(partyBlock), int(cellX), int(cellY))
	mzRedrawView = 1
	if f := getBitfield(mazeA.Floor[:], c, 4); (f == 1 || f == 2) && getBitfield(mazeA.Feature[:], c, 2) == dir {
		mzRedrawView = 0
	}
	stepPosition(dir, 1)
	if moveToMazeCell(&cellX, &cellY, &partyBlock, int(partyX), int(partyY)) != 0 {
		if gameStateArg != 10 && gameStateArg != 12 {
			killParty(2022)
			return 1
		}
		restorePosition()
		return 0
	}
	c = cellIndex(int(partyBlock), int(cellX), int(cellY))
	pit := testBit(mazeA.Bits1082[:], c)
	b2 := testBit(mazeA.Bits1178[:], c)
	if partyCounters[0].B == 1 && !pit {
		partyCounters[0].B = 0
	}
	if pit {
		switch gameStateArg {
		case 0, 2, 3, 4, 6, 11:
			killParty(10000)
			return 1
		case 1:
			if partyBlock != 10 {
				killParty(10000)
				return 1
			}
			redrawMazeView()
			playSound(8)
			if cellX < 4 {
				partyBlock = 4
				cellX += 4
			} else {
				partyBlock = 5
				cellX -= 4
			}
			cellY += 3
			partyX = int16(mazeA.BlockX[partyBlock]) + cellX
			partyY = int16(mazeA.BlockY[partyBlock]) + cellY
			moveToMazeCell(&cellX, &cellY, &partyBlock, int(partyX), int(partyY))
			b2 = false
			savePosition()
		case 8, 10, 12:
			if !(gameStateArg == 12 && partyBlock >= 9 || partyCounters[0].B != 0) {
				restorePosition()
				return 0
			}
		}
	}
	if b2 && gameStateArg == 2 {
		killParty(2022)
		return 1
	}
	cellEventIndex = int16(mazeB.FirstEvent[partyBlock])
	return 0
}

// move_party (WMAZE 5949) moves the party one step towards facing+turn, handling walls,
// doors and the current cell's event direction.
func moveParty(turn int) {
	mzG503a = 2020
	dir := (int(partyFacing) + turn + 4) % 4
	wall := getWallInDirection(dir, int(partyBlock), int(cellX), int(cellY), int(partyX), int(partyY))
	bump := func() {
		snd := 0
		if mzG503a == 2020 {
			snd = 2
		}
		showMessageHeld(int(mzG503a), snd, 2)
	}
	switch curEvent[dir] {
	case 0:
		if wall < 2 {
			if tryMove(dir) != 0 {
				return
			}
		} else if wall == 3 && dir == int(partyFacing) {
			playSound(4)
			g363e = 1
			redrawMazeView()
			playSound(13)
			waitSecondsOrEnter(2)
			g363e = 2
			redrawMazeView()
			waitSecondsOrEnter(2)
			g363e = 0
			if tryMove(dir) != 0 {
				return
			}
		}
		if mzG5034 == 0 {
			bump()
		}
	case 1, 2:
		if wall == 3 && dir == int(partyFacing) {
			showMessageHeld(2021, 2, messageDelay/2)
		} else {
			showMessageHeld(int(mzG503a), 2, 2)
		}
	case 3:
		if wall == 2 {
			tryMove(dir)
		} else {
			showMessageHeld(int(mzG503a), 2, 2)
		}
	}
}

// menuCell is where entry i of a menu laid out in columns of rows entries, width apart,
// from (x, y), sits.
func menuCell(i, width, rows, x, y int) (int, int) { return (i/rows)*width + x, i%rows + y }

// f5fa1 (WMAZE): swaps entries a and b of order and redraws both names in win.
func swapOrderEntries(win *Window, a, b int, order []int16) {
	order[a], order[b] = order[b], order[a]
	for _, i := range []int{a, b} {
		x, y := memberCell(i)
		setCursor(win, x, y)
		printStringHi(win, memberName(int(order[i])), 3)
	}
}
