package game

import "strings"

// WMAZE cell events (doors, walls, flags, puzzles, riddles, member effects), the event
// dispatcher, searching and the locked-door mini-games (WMAZE b53d..db1e).

// byteAt is s[i] as the C reads a char buffer: 0 past the end of the string.
func byteAt(s string, i int) int {
	if i < 0 || i >= len(s) {
		return 0
	}
	return int(s[i])
}

// wallChangeEvent (WMAZE b53d): event 21, open or close a wall or door somewhere in the map,
// animating it when the party faces it.
func wallChangeEvent() {
	ev := int(cellEventIndex)
	bits := int(mazeB.DirBits[ev])
	doorA := int(mazeB.Data[ev]) / 256
	doorB := int(mazeB.Data[ev]) % 256
	side := int(mazeB.Unk288[ev]) / 256
	face := int(mazeB.Unk288[ev]) % 256
	rel := (int(partyFacing) - side + 4) % 4
	if rel == 3 {
		rel = -1
	}
	snd := int(mazeB.Param[ev])
	x := int16(extractBits(bits, 0, 3))
	y := int16(extractBits(bits, 3, 3))
	blk := int16(extractBits(bits, 6, 4))
	door := extractBits(bits, 10, 1)
	closed := extractBits(bits, 11, 1)
	anim := extractBits(bits, 12, 1)
	mx := int(mazeA.BlockX[blk]) + int(x)
	my := int(mazeA.BlockY[blk]) + int(y)
	val := 1
	if closed != 0 {
		val = 2
	}
	cell := int(blk)<<6 + int(y)<<3 + int(x)
	if anim != 0 && door == 1 && rel == 0 && closed == 1 {
		if int(partyFacing) == face {
			animateDoor(0, doorA, doorB, snd)
		} else {
			shakeView(7, snd)
		}
	}
	switch side {
	case 0:
		setBitfield(mazeA.Walls0[:], cell, 2, val)
		if door == 1 {
			setDoorFeature(mx, my, 0, closed, 0)
			setDoorFeature(mx, my+1, 2, closed, 0)
		}
	case 1:
		setBitfield(mazeA.Walls1[:], cell, 2, val)
		if door == 1 {
			setDoorFeature(mx, my, 1, closed, 0)
			setDoorFeature(mx+1, my, 3, closed, 0)
		}
	case 2:
		if door == 1 {
			setDoorFeature(mx, my, 2, closed, 0)
			setDoorFeature(mx, my-1, 0, closed, 0)
		}
		if moveToMazeCell(&x, &y, &blk, mx, my-1) == 0 {
			cell = int(blk)<<6 + int(y)<<3 + int(x)
			setBitfield(mazeA.Walls0[:], cell, 2, val)
		}
	case 3:
		if door == 1 {
			setDoorFeature(mx, my, 3, closed, 0)
			setDoorFeature(mx-1, my, 1, closed, 0)
		}
		if moveToMazeCell(&x, &y, &blk, mx-1, my) == 0 {
			cell = int(blk)<<6 + int(y)<<3 + int(x)
			setBitfield(mazeA.Walls1[:], cell, 2, val)
		}
	}
	switch {
	case anim == 0:
		if door == 1 {
			shakeView(7, 3)
		} else {
			playSound(4)
		}
		waitAnimating(2)
	case door != 1:
		redrawMazeView()
		if snd > 0 {
			playSound(snd)
		}
	case rel == 0 && closed == 0:
		if int(partyFacing) == face {
			animateDoor(1, doorA, doorB, snd)
		} else {
			shakeView(7, snd)
		}
	case rel == 1 && closed == 0 || rel == -1 && closed == 1:
		drawMazeView()
		shakeView(1, snd)
		drvShowWindow(pictureWin)
		shakeView(6, snd)
	case rel == 1 && closed == 1 || rel == -1 && closed == 0:
		drawMazeView()
		shakeView(6, snd)
		drvShowWindow(pictureWin)
		shakeView(1, snd)
	case rel == 2:
		shakeView(7, snd)
	}
	if mazeB.FlagBit[ev] != 0 {
		setEventFlags(2)
	}
}

// clearFlagEvent (WMAZE b9a2): event 22, clear one of two level flags by chance.
func clearFlagEvent() {
	waitAnimating(5)
	ev := int(cellEventIndex)
	bits := int(mazeB.DirBits[ev])
	n := extractBits(bits, 0, 8)
	var odds int
	if p := int(mazeB.Param[ev]); p > 0 {
		odds = p
	} else {
		odds = random(n)
	}
	flag := int(mazeB.Unk288[ev])
	if random(n) < odds {
		flag = int(mazeB.Data[ev])
	}
	setBit(levelFlags[gameStateArg][:], flag, false)
	if extractBits(bits, 8, 1) == 1 {
		setBit(scenarioHdr.LevelFlags[gameStateArg][:], flag, false)
	}
	if gameStateArg == 7 && flag == 24 {
		setBit(levelFlags[gameStateArg][:], 18, true)
		setBit(scenarioHdr.LevelFlags[gameStateArg][:], 18, true)
	}
	setEventFlags(extractBits(int(mazeB.DirBits[ev]), 9, 2))
}

// changeFloorEvent (WMAZE baed): event 23, change a cell's floor and feature.
func changeFloorEvent() {
	ev := int(cellEventIndex)
	bits := int(mazeB.DirBits[ev])
	cell := extractBits(bits, 6, 4)<<6 + extractBits(bits, 3, 3)<<3 + extractBits(bits, 0, 3)
	setBitfield(mazeA.Floor[:], cell, 4, int(mazeB.Data[ev]))
	setBitfield(mazeA.Feature[:], cell, 2, int(mazeB.Param[ev]))
	if extractBits(bits, 10, 1) == 1 {
		redrawMazeView()
	}
	if mazeB.Unk288[ev] > 0 {
		playSound(int(mazeB.Unk288[ev]))
	}
	setEventFlags(extractBits(bits, 11, 2))
}

// swapLevelFlags (WMAZE bbe7) clears level flag a and sets b (the other way round with
// swap), in the game and in the scenario header; flag 0 is none.
func swapLevelFlags(a, b int, swap bool) {
	clr, set := a, b
	if swap {
		clr, set = b, a
	}
	if clr != 0 {
		setBit(levelFlags[gameStateArg][:], clr, false)
		setBit(scenarioHdr.LevelFlags[gameStateArg][:], clr, false)
	}
	if set != 0 {
		setBit(levelFlags[gameStateArg][:], set, true)
		setBit(scenarioHdr.LevelFlags[gameStateArg][:], set, true)
	}
}

// puzzleEvent (WMAZE bc93): event 12, a multiple-choice puzzle; the right sequence of
// answers swaps two level flags.
func puzzleEvent() {
	ev := int(cellEventIndex)
	if mazeB.Param[ev] != 0 {
		showMessageText(int(mazeB.Data[ev])+int(mazeB.Param[ev]), 0)
	}
	bits := int(mazeB.DirBits[ev])
	choices := extractBits(bits, 0, 4)
	steps := extractBits(bits, 4, 4)
	snd := extractBits(bits, 8, 4)
	anyOrder := extractBits(bits, 12, 1)
	single := extractBits(bits, 13, 1)
	once := extractBits(bits, 14, 1)
	abort := extractBits(bits, 15, 1)
	eventScreenDirty = 1
	msg := int(mazeB.Data[ev])
	eraseWindow(mainWindow, false)
	deferWindowDraw(mainWindow)
	var answer string
	if single == 1 {
		answer = getMessage(2038)
	} else {
		answer = getMessage(msg)
		msg++
	}
	setCursor(mainWindow, 20-(len(answer)+1)/2, 0)
	printStringHi(mainWindow, answer, 3)
	resumeWindowDraw(mainWindow)
	textWin = createWindow(0, 19, 40, 5, 20, -1, 0, false)
	fillWindow(textWin, 32, 3)
	eraseWindow(bottomLineWin, false)
	refreshAllWindows()
	answer = getMessage(msg)
	msg++
	enabled := make([]int, choices+1)
	for i := range choices {
		enabled[i] = 1
	}
	enabled[choices] = -1
	if single == 1 {
		t := getMessage(msg)
		c := menuSelectCancel(textWin, msg, enabled, 20-(len(t)+1)/2, 1, 20, 4, 5, 0, 0, func() int { return getMazeInput(mzMsg2001) })
		if f := byteAt(answer, c<<1)*10 + byteAt(answer, c<<1+1) - 528; f > 0 && f < 80 {
			setBit(levelFlags[gameStateArg][:], f, false)
		}
	} else {
		failed, gaveUp := false, false
		for i := 0; i < steps; i++ {
			fillWindow(textWin, 32, 3)
			refreshWindow(textWin)
			t := getMessage(msg)
			c := menuSelectCancel(textWin, msg, enabled, 20-(len(t)*2+4)/2, 1, len(t)+3, (choices-1)/2+1, 5, 0, 0, func() int { return getMazeInput(mzMsg2001) })
			if once == 1 && c >= 0 {
				enabled[c] = 0
			}
			if choices-1 == c { // the last choice leaves the puzzle
				gaveUp = !failed
				break
			}
			if snd > 0 {
				playSound(snd)
			}
			if strings.IndexByte(answer, byte(c+48)) != -1 && (anyOrder != 0 || byteAt(answer, i)-48 == c) {
				continue
			}
			failed = true
			if abort == 1 {
				break
			}
		}
		if !gaveUp {
			u := int(mazeB.Unk288[ev])
			swapLevelFlags(extractBits(u, 0, 8), extractBits(u, 8, 8), failed)
		}
	}
	eraseWindow(mainWindow, false)
	destroyWindow(textWin, false)
	eraseCursorMark(bottomLineWin, false)
}

// checkPartyItemEvent (WMAZE c0bf): event 6, swap two level flags on whether a member
// carries the item (flag bit 0 set).
func checkPartyItemEvent() {
	ev := int(cellEventIndex)
	missing := true
	id := mazeB.Data[ev]
	for m := 0; m < partyCount; m++ {
		pc := &party[m]
		for i := 0; i < int(pc.ItemCount); i++ {
			if pc.Items[i].ID == id && pc.Items[i].Flags&1 != 0 {
				missing = false
			}
		}
	}
	swapLevelFlags(int(mazeB.Unk288[ev]), int(mazeB.Param[ev]), missing)
	if !missing {
		setEventFlags(extractBits(int(mazeB.DirBits[ev]), 0, 2))
	}
}

// riddleEvent (WMAZE c182): event 13, ask for a word; a message starting with '$' is
// accepted anywhere in the reply, others must match it whole.
func riddleEvent() {
	ev := int(cellEventIndex)
	bits := int(mazeB.DirBits[ev])
	if extractBits(bits, 11, 1) == 1 {
		g4f8a = int16(extractBits(bits, 9, 2))
		if g4f8a != partyFacing {
			return
		}
	}
	showMessageText(int(mazeB.Data[ev]), 0)
	flags := extractBits(bits, 8, 1)
	m := extractBits(bits, 0, 8) + int(mazeB.Data[ev])
	deferWindowDraw(mainWindow)
	deferWindowDraw(bottomLineWin)
	textWin = createWindow(0, 19, 40, 5, 20, -1, 0, false)
	refreshAllWindows()
	setCursor(textWin, 2, 0)
	printMessage(2039, textWin, 1)
	reply := mzInputText(textWin, 20, 6, 0)
	wrong := true
	for acc := getMessage(m); !msgNotFound; acc = getMessage(m) {
		if strings.HasPrefix(acc, "$") {
			if strings.Contains(reply, acc[1:]) {
				wrong = false
			}
		} else if reply == acc {
			wrong = false
		}
		m++
	}
	swapLevelFlags(int(mazeB.Unk288[ev]), int(mazeB.Param[ev]), wrong)
	if !wrong {
		setEventFlags(flags + 1)
	}
	destroyWindow(textWin, false)
	resumeWindowDraw(mainWindow)
	resumeWindowDraw(bottomLineWin)
}

// swapFlagsEvent (WMAZE c363): event 24, swap two level flags on a scenario header bit.
func swapFlagsEvent() {
	ev := int(cellEventIndex)
	set := testBit(scenarioHdr.Unk140[20:], int(mazeB.Data[ev])) // DS:0x3498
	swapLevelFlags(int(mazeB.Unk288[ev]), int(mazeB.Param[ev]), set)
	setEventFlags(extractBits(int(mazeB.DirBits[ev]), 0, 2))
}

// clampValue (WMAZE c3c7) is v clamped into lo..hi.
func clampValue(v, lo, hi int) int {
	v = int(int16(v)) // the C's int argument wraps
	if lo > v {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// showCenteredNotice (WMAZE c3e5) shows message msg centred in the main window.
func showCenteredNotice(msg int) {
	deferWindowDraw(mainWindow)
	eraseWindow(mainWindow, false)
	eraseCursorMark(mainWindow, false)
	s := getMessage(msg)
	setCursor(mainWindow, 10-(len(s)+1)/2, 0)
	printStringHi(mainWindow, s, 3)
	resumeWindowDraw(mainWindow)
	refreshWindow(mainWindow)
	playSound(0)
	waitAnimating(messageDelay)
}

// applyMemberEffects (WMAZE c477): event 25, a fountain or the like: each chosen member
// gets the effects of the event's bits (0..6 good, 8..14 bad).
func applyMemberEffects() {
	ev := int(cellEventIndex)
	g4f8a = int16(mazeB.Param[ev])
	if g4f8a != partyFacing {
		return
	}
	if mazeB.Data[ev] > 0 {
		showMessageText(int(mazeB.Data[ev]), 0)
	}
	bits := int(mazeB.DirBits[ev])
	count := int(mazeB.Unk288[ev]) / 16
	sides := int(mazeB.Unk288[ev]) % 16
	for {
		m := choosePartyMember(2047, 1)
		if m == -1 {
			break
		}
		pc := &party[m]
		good, bad := 0, 0
		if pc.Condition < 3 {
			roll := rollDice(formatDice(count, sides, count))
			for b := 0; b < 16; b++ {
				if extractBits(bits, b, 1) != 1 {
					continue
				}
				if b < 7 {
					good = 1
				}
				if b > 7 && b < 15 {
					bad = 2
				}
				switch b {
				case 0:
					pc.Stamina = int16(clampValue(int(pc.Stamina)+roll, 0, int(pc.StaminaMax)))
				case 1:
					for r := range 6 {
						sp := &pc.SP[r]
						sp.Cur = int16(clampValue(rollDice(formatDice(count, sides, count))+int(sp.Cur), 0, int(sp.Max)))
					}
				case 2:
					pc.HP = int16(clampValue(int(pc.HP)+roll, 0, int(pc.HPMax)))
				case 3:
					pc.PoisonStr = 0
					pc.Status[5] = 0
				case 4:
					pc.Status[3] = 0
				case 5:
					if pc.Condition == 2 {
						pc.Condition = 0
					}
				case 6:
					for _, s := range []int{1, 2, 4, 8, 9} {
						pc.Status[s] = 0
					}
				case 8:
					pc.Stamina = int16(clampValue(int(pc.Stamina)-roll, 0, int(pc.StaminaMax)))
					if pc.Stamina <= 0 {
						pc.Status[2] += uint8(random(6) + 6)
					}
				case 9:
					for r := range 6 {
						sp := &pc.SP[r]
						sp.Cur = int16(clampValue(int(sp.Cur)-rollDice(formatDice(count, sides, count)), 0, int(sp.Max)))
					}
				case 10:
					pc.HP = int16(clampValue(int(pc.HP)-roll, 0, int(pc.HPMax)))
					if pc.HP <= 0 {
						pc.Condition = 3
						pc.HP = 0
						pc.Stamina = 0
						playSound(int(pc.Sex) + 8)
					}
				case 11:
					if int(pc.PoisonStr) < count {
						pc.PoisonStr = uint8(count)
					}
					if int(pc.Status[5]) < roll {
						pc.Status[5] = uint8(roll)
					}
				case 12:
					if int(pc.Status[3]) < roll {
						pc.Status[3] = uint8(roll)
					}
				case 13:
					if pc.Condition < 2 {
						pc.Condition = 2
					}
				case 14:
					if int(pc.Status[2]) < roll {
						pc.Status[2] = uint8(roll)
					}
				}
			}
		}
		drawMemberStatus(m)
		showCenteredNotice(good + bad + 2048)
	}
	refreshAllWindows()
}

// runCellEvents (WMAZE c8eb) runs each event record of the party's cell while the game
// stays in the maze.
func runCellEvents() {
	g4f8a = -1
	eventScreenDirty = 0
	for {
		cellEventIndex = int16(findCellEvent(int(cellX), int(cellY), int(partyBlock), int(cellEventIndex)))
		if cellEventIndex == -1 {
			break
		}
		mzG5232 = 1
		switch mazeB.Kind[cellEventIndex] {
		case 1:
			doTeleportEvent()
		case 2:
			if mzG5034 != 0 {
				spinnerEvent()
			}
		case 3:
			doMessageEvent()
		case 4:
			encounterEvent()
		case 5:
			npcEvent()
		case 6:
			checkPartyItemEvent()
		case 7:
			setDirValuesEvent()
		case 8:
			if mazeB.Data[cellEventIndex] > 0 {
				showMessageText(int(mazeB.Data[cellEventIndex]), 0)
			} else {
				checkSecretDetect()
			}
		case 9:
			trapEvent()
		case 10:
			changeWallsEvent()
		case 11:
			setFlagEvent()
		case 12:
			puzzleEvent()
		case 13:
			riddleEvent()
		case 16:
			textEvent()
		case 17:
			wmazeFa68d()
		case 18:
			wmazeFa385()
		case 21:
			wallChangeEvent()
		case 22:
			clearFlagEvent()
		case 23:
			changeFloorEvent()
		case 24:
			swapFlagsEvent()
		case 25:
			applyMemberEffects()
		}
		if gameState != 5 {
			return
		}
		if mzG5232 == 1 {
			cellEventIndex++
		}
	}
	if eventScreenDirty == 1 {
		refreshAllWindows()
	}
}

// clearDirBits (WMAZE ca32) clears bits bit and bit+1 of event ev's direction bits.
func clearDirBits(ev, bit int) {
	mazeB.DirBits[ev] &^= 3 << bit
}

// revealHiddenDoor (WMAZE ca72): searching found a hidden door in front of the party; put it
// into the walls.
func revealHiddenDoor(ev int) {
	x, y, blk := cellX, cellY, partyBlock
	mx := int(mazeA.BlockX[blk]) + int(x)
	my := int(mazeA.BlockY[blk]) + int(y)
	cell := int(blk)<<6 + int(y)<<3 + int(x)
	switch partyFacing {
	case 0:
		setBitfield(mazeA.Walls0[:], cell, 2, 1)
	case 1:
		setBitfield(mazeA.Walls1[:], cell, 2, 1)
	case 2:
		if moveToMazeCell(&x, &y, &blk, mx, my-1) == 0 {
			setBitfield(mazeA.Walls0[:], int(blk)<<6+int(y)<<3+int(x), 2, 1)
		}
	case 3:
		if moveToMazeCell(&x, &y, &blk, mx-1, my) == 0 {
			setBitfield(mazeA.Walls1[:], int(blk)<<6+int(y)<<3+int(x), 2, 1)
		}
	}
	setDoorFeature(mx, my, 0, 0, 1)
	redrawMazeView()
	playSound(4)
	clearDirBits(ev, int(partyFacing)<<1+8)
}

// tryFlagEventAt (WMAZE cc10) sets the level flag of flag event ev if it applies to the
// party's facing; whether it did.
func tryFlagEventAt(ev int) bool {
	bits := int(mazeB.DirBits[ev])
	if extractBits(bits, 14, 1) == 1 && extractBits(bits, 12, 2) != int(partyFacing) {
		return false
	}
	setBit(levelFlags[gameStateArg][:], extractBits(bits, 0, 8), extractBits(bits, 10, 1) != 0)
	cellEventIndex = int16(ev)
	setEventFlags(extractBits(bits, 8, 2))
	cellEventIndex = int16(ev + 1)
	mzG5034 = 1
	return true
}

// searchCell (WMAZE ccd2) searches the party's cell: a secret feature ahead, hidden doors
// (event 7) and hidden flags (event 8).
func searchCell() {
	cell := int(partyBlock)<<6 + int(cellY)<<3 + int(cellX)
	nothing := true
	if getBitfield(mazeA.Floor[:], cell, 4) == 5 && getBitfield(mazeA.Feature[:], cell, 2) == int(partyFacing) {
		showMazeMessage(2044)
		playSound(4)
		waitAnimating(2)
		nothing = false
	} else {
		showMazeMessage(2043)
		waitAnimating(messageDelay)
	}
	for ev := int(mazeB.FirstEvent[partyBlock]); ; ev++ {
		if ev = findCellEvent(int(cellX), int(cellY), int(partyBlock), ev); ev == -1 {
			break
		}
		k := mazeB.Kind[ev]
		if k == 7 && extractBits(int(mazeB.DirBits[ev]), int(partyFacing)<<1+8, 2) == 2 {
			revealHiddenDoor(ev)
			return
		}
		if k == 8 && tryFlagEventAt(ev) {
			return
		}
	}
	if nothing {
		showMazeMessage(2045)
		playSound(0)
		waitAnimating(messageDelay / 2)
	}
	eraseWindow(mainWindow, true)
}

// jamLock (WMAZE ce13) sets lock event ev's state for the party's facing to 2 (jammed).
func jamLock(ev int) {
	b := int(partyFacing) << 1
	mazeB.DirBits[ev] = mazeB.DirBits[ev]&^(1<<b) | 1<<(b+1)
}

// openLockAhead (WMAZE ce81): the lock ahead opens.
func openLockAhead(ev int) {
	curEvent[partyFacing] = 0
	clearDirBits(ev, int(partyFacing)<<1)
	playSound(4)
}

// lockFrame draws the box of the lock mini-games, right edge at column right.
func lockFrame(win *Window, right int) {
	for _, g := range [][3]int{{0, 17, 19}, {1, 4, 5}, {2, 9, 10}, {3, 4, 5}, {4, 6, 8}} {
		setFrameGlyph(win, 0, g[0], byte(g[1]))
		setFrameGlyph(win, right, g[0], byte(g[2]))
	}
	for c := 1; c < right; c++ {
		setFrameGlyph(win, c, 0, 18)
		setFrameGlyph(win, c, 2, 12)
		setFrameGlyph(win, c, 4, 7)
	}
}

// lockResult shows message 2113+result in the main window and closes win.
func lockResult(win *Window, result int) {
	eraseWindow(mainWindow, true)
	s := getMessage(result + 2113)
	setCursor(mainWindow, 20-(len(s)+1)/2, 0)
	printStringHi(mainWindow, s, 3)
	waitAnimating(messageDelay)
	destroyWindow(win, false)
}

// forceLock (WMAZE ced8): the strength bar mini-game for forcing lock event ev with
// member; 0 on success, 1 failed, 2 jammed (or the member gave out: jammed+1).
func forceLock(member, ev int) int {
	clearClickFlag()
	win := createWindow(20, 19, 20, 5, 20, -1, 0, false)
	fillWindow(win, 32, 1)
	lockFrame(win, 19)
	diff := extractBits(int(mazeB.Param[ev]), 0, 5)
	jammed := 0
	if extractBits(int(mazeB.DirBits[ev]), int(partyFacing)<<1, 2) == 2 {
		jammed = 1
	}
	setCursor(mainWindow, 21, 0)
	printStringHi(mainWindow, getMessage(2112), 3)
	pc := &party[member]
	target := 18 - (int(pc.Stats[0]) - diff) + diff
	target = max(1, min(target, 18))
	if jammed != 0 {
		target = 18
	}
	deferWindowDraw(win)
	setCursor(win, 1, 3)
	for range target {
		putChar(win, 'a', 12)
	}
	resumeWindowDraw(win)
	going := true
	strength := 0
	for {
		deferWindowDraw(win)
		power := int(int16(int32(pc.Stamina) * 100 / int32(pc.StaminaMax) * int32(pc.Stats[0]) / 100)) // (int) cast
		if power <= 0 || random(50) == 0 {
			pc.Stamina--
			if power <= 0 || pc.Stamina <= 0 {
				pc.Status[2] = uint8(random(6) + 6)
				pc.Stamina = 0
				drawMemberStatus(member)
				destroyWindow(win, false)
				return jammed + 1
			}
			drawMemberStatus(member)
		}
		strength = int(int16(random(power)+random(power)+random(power)+random(power))) / 4 // C int sum wraps
		strength = max(1, min(strength, 18))
		setCursor(win, 1, 1)
		color := 6
		if jammed == 1 {
			color = 4
		}
		for range strength {
			putChar(win, 'a', color)
		}
		clearWindowArea(win, int(win.CurX), 1, 18, 1)
		resumeWindowDraw(win)
		refreshWindow(win)
		if keyAvailable(1) && int8(getKey(1)) == 13 {
			going = false
		}
		if !going || checkScreenClick() {
			break
		}
	}
	result := 1
	if strength >= target {
		result = 0
	}
	if jammed != 0 {
		result = 2
	}
	if result != 0 {
		if random(3) == 0 {
			jamLock(ev)
		}
	} else {
		openLockAhead(ev)
	}
	lockResult(win, result)
	return result
}

// pickLock (WMAZE d3b3): the tumbler mini-game for picking lock event ev with member;
// trains skill 15 for thieves, ninjas and bards. 0 on success.
func pickLock(member, ev int) int {
	clearClickFlag()
	win := createWindow(20, 19, 20, 5, 20, -1, 0, false)
	fillWindow(win, 32, 1)
	lockFrame(win, 18)
	for c := 3; c < 18; c += 3 {
		for row, g := range []byte{34, 13, 16, 13, 11} {
			setFrameGlyph(win, c, row, g)
		}
	}
	deferWindowDraw(win)
	for r := range 5 {
		setCursor(win, 19, r)
		putCharHi(win, ' ', 3)
	}
	resumeWindowDraw(win)
	diff := extractBits(int(mazeB.Param[ev]), 0, 5)
	pc := &party[member]
	skill := min(int(pc.Level)+int(pc.Skills[15]), 95)
	tumblers := min(diff/3+1, 6)
	jammed := 0
	if extractBits(int(mazeB.DirBits[ev]), int(partyFacing)<<1, 2) == 2 {
		lockTumblers[(int(partyX)+int(partyY))%tumblers] = 100
		jammed = 1
	}
	setCursor(mainWindow, 21, 0)
	printStringHi(mainWindow, getMessage(2111), 3)
	going := true
	var result int
	for {
		result = 0
		deferWindowDraw(win)
		for t := range tumblers {
			roll := random(skill)
			setCursor(win, t*3+1, 3)
			printInt(win, roll, 2, 1)
			setCursor(win, t*3+1, 1)
			color := 6
			if roll <= int(lockTumblers[t]) {
				color = 12
				result = jammed + 1
			}
			printString(win, "cc", color)
		}
		resumeWindowDraw(win)
		refreshWindow(win)
		if keyAvailable(1) && int8(getKey(1)) == 13 {
			going = false
		}
		if !going || checkScreenClick() {
			break
		}
	}
	thief := pc.Class == 3 || pc.Class == 6 || pc.Class == 13
	if result != 0 {
		if result == 1 && thief {
			trySkillGain(member, 15, 1)
		}
		if random(3) == 0 {
			jamLock(ev)
		}
	} else {
		openLockAhead(ev)
		if thief {
			trySkillGain(member, 15, 4)
		}
	}
	lockResult(win, result)
	return result
}

// lockedDoorMenu (WMAZE d8a9): force or pick lock event ev until it opens or the player
// leaves.
func lockedDoorMenu(ev int) {
	diff := extractBits(int(mazeB.Param[ev]), 0, 5)
	for i := range lockTumblers {
		lockTumblers[i] = int16(random(diff * 5))
	}
	enabled := []int{1, 1, 1, -1} // the C's l20 (-1) ends the list
	deferWindowDraw(g4fb0)
	win := createWindow(0, 19, 19, 5, 10, -4, 0, false)
	for {
		fillWindow(win, 32, 3)
		eraseWindow(mainWindow, false)
		eraseCursorMark(mainWindow, false)
		setCursor(mainWindow, 3, 0)
		deferWindowDraw(mainWindow)
		printMessage(2002, mainWindow, 18)
		resumeWindowDraw(mainWindow)
		refreshAllWindows()
		choice := menuSelectCancel(win, 2100, enabled, 1, 1, 7, 1, 5, 0, 0, func() int { return getMazeInput(mzMsg2001) })
		switch choice {
		case 0, 1:
			game := forceLock
			if choice == 1 {
				game = pickLock
			}
			if m := choosePartyMember(2110, 1); m != -1 && isMemberAble(m) && game(m, ev) == 0 {
				choice = 2
			}
		}
		if choice == 2 {
			break
		}
	}
	destroyWindow(win, false)
	resumeWindowDraw(g4fb0)
	eraseWindow(mainWindow, false)
}

// checkLockedDoorAt (WMAZE da96): a message event ev that is a locked chest opens the
// treasure (game state 21); whether it did.
func checkLockedDoorAt(ev int) bool {
	bits := int(mazeB.DirBits[ev])
	if extractBits(bits, 14, 1) == 1 && extractBits(bits, 12, 2) != int(partyFacing) {
		return false
	}
	if extractBits(bits, 8, 1) == 0 {
		return false
	}
	cellEventIndex = int16(ev)
	treasureID = int16(mazeB.Param[ev])
	gameState = 21
	return true
}

// handleLockedDoor (WMAZE db1e): the party tries the door or chest ahead.
func handleLockedDoor() {
	for ev := int(mazeB.FirstEvent[partyBlock]); ; ev++ {
		if ev = findCellEvent(int(cellX), int(cellY), int(partyBlock), ev); ev == -1 {
			return
		}
		if mazeB.Kind[ev] == 3 && checkLockedDoorAt(ev) {
			return
		}
		if mazeB.Kind[ev] == 7 {
			if s := extractBits(int(mazeB.DirBits[ev]), int(partyFacing)<<1, 2); s == 1 || s == 2 {
				lockedDoorMenu(ev)
				return
			}
		}
	}
}
