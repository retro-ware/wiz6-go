package game

import (
	"os"

	"github.com/retro-ware/wiz6-go/internal/dbs"
)

// WMAZE: the party menus, resting, saving, the main loop, stepping and fleeing.

// partyCell is the party's cell index (block<<6 | y<<3 | x).
func partyCell() int { return int(partyBlock)<<6 | int(cellY)<<3 | int(cellX) }

// startEncounter rolls a random encounter for the party's block and enters combat
// (state 10).
func startEncounter(g4338v int16) {
	g433a = wmazeF467b(wmazeF46e8(int(partyBlock)))
	g4338 = g4338v
	g4354 = 0
	gameState = 10
}

// reorder_party (WMAZE 606f) lets the player pick the members in their new order.
func reorderParty() {
	n := partyCount
	var saved [6]PC
	var pcfiles [6]int
	var portraits [6]uint8
	for k := 0; n-1 > k; k++ {
		curMember = int16(choosePartyMember(2037, 0))
		c := int(curMember)
		saved[k] = party[c]
		pcfiles[k] = memberPcfileIndex[c]
		portraits[k] = memberPortrait[c]
		partyCount--
		for l := c; l < partyCount; l++ {
			party[l] = party[l+1]
			memberPcfileIndex[l] = memberPcfileIndex[l+1]
			memberPortrait[l] = memberPortrait[l+1]
		}
	}
	saved[n-1] = party[0]
	pcfiles[n-1] = memberPcfileIndex[0]
	portraits[n-1] = memberPortrait[0]
	partyCount = n
	for l := 0; l < partyCount; l++ {
		party[l] = saved[l]
		memberPcfileIndex[l] = pcfiles[l]
		memberPortrait[l] = portraits[l]
		drawMemberStatus(l)
	}
}

// heal_member (WMAZE 61f8) regenerates member m: hp by random(hpRoll) plus regeneration
// less drain and poison (times rate), stamina by staminaPct percent, status durations
// down by rate, and with spells the spell points.
func healMember(m, hpRoll, staminaPct, rate, spells int) {
	pc := &party[m]
	if pc.Condition >= 3 {
		return
	}
	pc.HP += int16(random(hpRoll) + int(pc.Regen)*rate - int(pc.Drain)*rate - int(pc.PoisonStr)*rate)
	if pc.HP > pc.HPMax {
		pc.HP = pc.HPMax
	}
	if pc.HP <= 0 {
		pc.HP = 0
		pc.Stamina = 0
		pc.Condition = 3
		playSound(int(pc.Sex) + 8)
		drawMemberStatus(m)
		return
	}
	if pc.Stamina <= 0 {
		pc.Stamina = 0
		pc.Status[2] = uint8(random(6) + 6)
	}
	if staminaPct > 0 {
		n := int(int16(int32(pc.StaminaMax) * int32(staminaPct) / 100))
		pc.Stamina += int16(random(n/2) + n/2 + 1)
		if pc.Stamina > pc.StaminaMax {
			pc.Stamina = pc.StaminaMax
		}
	}
	for i := range 10 {
		if v := int(pc.Status[i]); v > 0 && v != 255 {
			v -= rate
			if v < 0 {
				v = 0
			}
			pc.Status[i] = uint8(v)
		}
	}
	if pc.Status[5] == 0 || pc.PoisonStr == 0 {
		pc.Status[5] = 0
		pc.PoisonStr = 0
	}
	if spells > 0 {
		for r := range 6 {
			if pc.SpellRegen[r] < 1 {
				pc.SpellRegen[r] = 1
			}
			sp := &pc.SP[r]
			sp.Cur += int16(random(int(pc.SpellRegen[r]) + 1))
			if sp.Cur > sp.Max {
				sp.Cur = sp.Max
			}
		}
	}
	drawMemberStatus(m)
}

// rest_party (WMAZE 63f4) rests the party for up to eight hours; true if an encounter
// interrupts it.
func restParty() bool {
	showMazeMessage(2036)
	mzG5038 = 0
	hours := 8
	s := &saveState
	s.G4f78 += s.G4f80 - s.G4f7c
	n := int(int16(s.G4f78 / 60))
	full := 0 // hours of full rest banked
	for n >= 4 {
		full++
		n -= 4
		s.G4f78 -= 240
	}
	s.G4f7c = s.G4f80
	for hours > 0 && mzG5038 == 0 && gameState == 5 {
		for m := 0; m < partyCount; m++ {
			party[m].Status[2] = 10
			drawMemberStatus(m)
		}
		waitAnimating(10)
		hours--
		s.G4f80 += 60
		tickEffectTimers(60)
		for m := 0; m < partyCount; m++ {
			roll := 0
			if full > 0 {
				roll = int(party[m].Unk1A6) + 1
			}
			healMember(m, roll, int(party[m].Unk1A7), 6, 1)
			party[m].Status[2] = uint8(random(5))
			if party[m].PoisonStr > 0 {
				party[m].PoisonStr--
			}
		}
		full--
		party[random(partyCount)].Status[2] = 0
		safe := 975
		if hdr.Level == 1 {
			safe = 990
		}
		if random(1000) >= safe {
			return true
		}
		if keyAvailable(1) && int8(getKey(1)) == 13 {
			mzG5038 = 1
		}
	}
	for m := 0; m < partyCount; m++ {
		if int32(party[m].Stamina) > int32(party[m].StaminaMax)*3/4 {
			party[m].Status[2] = 0
		}
		drawMemberStatus(m)
	}
	return false
}

// is_member_able (WMAZE 65ea; also WPCVW, WTREA) is whether member m is well and awake;
// it beeps if not.
func isMemberAble(m int) bool {
	pc := &party[m]
	ok := pc.Condition == 0 && pc.Status[2] == 0 && pc.Status[3] == 0
	if !ok {
		playSound(0)
	}
	return ok
}

// show_error (WMAZE 6631) shows s with a beep.
func showError(s string) {
	showMazeText(s)
	playSound(0)
	waitAnimating(messageDelay)
}

// save_game (WMAZE 664f) writes SAVEGAME.DBS: the header, the 16 maps, the position block
// and the party; -1 on a failure. The floppy prompts of the C are no-ops in one
// directory and are left out.
func saveGame() int {
	done := getMessage(2257)
	failed := getMessage(2258)
	eraseWindow(mainWindow, false)
	stopSound()
	eraseCursorMark(bottomLineWin, false)
	resumeWindowDraw(g4fb0)
	refreshAllWindows()
	path := dbs.Path(dataDir, "SAVEGAME.DBS")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		showError(failed)
		return -1
	}
	defer f.Close()
	showMazeText(done)
	mapCacheA[gameStateArg] = *mazeA
	mapCacheB[gameStateArg] = *mazeB
	hdr.LevelFlags[gameStateArg] = levelFlags[gameStateArg]
	g := dbs.SaveGame{Hdr: hdr, MapA: mapCacheA, MapB: mapCacheB}
	for i := 14; i < 16; i++ { // the C writes zeros past the 14 maps
		g.MapA[i], g.MapB[i] = MapA{}, MapB{}
	}
	s := saveState
	s.Map, s.PartyX, s.PartyY = int16(gameStateArg), partyX, partyY
	s.CellX, s.CellY, s.Block, s.Facing = cellX, cellY, partyBlock, partyFacing
	s.Counters = partyCounters
	s.PartyCount = int16(partyCount)
	g.State = s
	g.Party = party[:partyCount]
	if _, err := f.Write(g.Encode()); err != nil {
		showError(failed)
		return -1
	}
	return 0
}

// disk_options_menu (WMAZE 6bcd): save, save and quit, quit, cancel, or the options.
func diskOptionsMenu() {
	win := createWindow(0, 19, 40, 5, 20, -1, 0, false)
	fillWindow(win, ' ', 3, false)
	eraseWindow(mainWindow, false)
	eraseWindow(bottomLineWin, false)
	refreshAllWindows()
	mzG503c = 1
	showMazeMessage(2200)
	// The C's l36[3] and l36[6] are its locals l30 (0) and l24 (-1, the end).
	enabled := []int16{1, 1, 1, 0, 1, 1, -1}
	choice := mazeMenuSelect(win, 2201, enabled, 1, 1, 20, 3, 5, 0)
	destroyWindow(win, false)
	switch choice {
	case 0:
		saveGame()
	case 1:
		if saveGame() != -1 {
			gameState = 7
		}
	case 2:
		gameState = 7
	case 4:
		gameState = 24
		returnCmd = 25
	}
	eraseCursorMark(bottomLineWin, false)
}

// maze_party_menu (WMAZE 6cf8): search, the member actions, locks, reorder, rest, disk.
func mazePartyMenu() {
	deferWindowDraw(g4fb0)
	win := createWindow(0, 19, 19, 5, 20, -4, 0, false)
	fillWindow(win, ' ', 3, false)
	eraseWindow(mainWindow, false)
	eraseCursorMark(mainWindow, false)
	refreshAllWindows()
	mzG503c = 1
	setCursor(mainWindow, 3, 0)
	printMessage(2002, mainWindow, 18)
	mainWindow.CurX++
	// The C's l36[9] is its local l18, -1, which ends the list.
	enabled := []int16{1, 1, 1, 1, 1, 1, 1, 1, 1, -1}
	if partyCount < 2 {
		enabled[5] = 0 // the C's l26 is enabled[5]
	}
	choice := mazeMenuSelect(win, 2005, enabled, 1, 1, 7, 3, 5, 0)
	destroyWindow(win, false)
	switch choice {
	case 0:
		searchCell()
		tickEffectTimers(1)
	case 1:
		if curMember = int16(choosePartyMember(2026, 1)); curMember != -1 {
			tickEffectTimers(1)
			gameState = 17
			returnCmd = 5
		}
	case 2:
		if curMember = int16(choosePartyMember(2029, 1)); curMember != -1 && isMemberAble(int(curMember)) {
			tickEffectTimers(1)
			gameState = 19
			returnCmd = 5
		}
	case 3:
		if curMember = int16(choosePartyMember(2031, 1)); curMember != -1 && isMemberAble(int(curMember)) && party[curMember].ItemCount > 0 {
			tickEffectTimers(1)
			gameState = 20
			returnCmd = 5
		}
	case 4:
		handleLockedDoor()
		tickEffectTimers(1)
	case 5:
		reorderParty()
	case 6:
		if restParty() {
			startEncounter(0)
		}
	case 7:
		diskOptionsMenu()
	}
	resumeWindowDraw(g4fb0)
	eraseWindow(mainWindow, false)
}

// f6f30 (WMAZE) puts four glyphs of font 2 as a 2x2 block at (x, y) of win.
func wmazeF6f30(win *Window, x, y int, a, b, c, d byte) {
	setCursor(win, x, y)
	putCharHi(win, a, 2)
	putCharHi(win, b, 2)
	setCursor(win, x, y+1)
	putCharHi(win, c, 2)
	putCharHi(win, d, 2)
}

// f6fa3 (WMAZE) puts glyphs first..first+7 of font 2 as a 4x2 block at (x, y) of win.
func wmazeF6fa3(win *Window, x, y int, first byte) {
	setCursor(win, x, y)
	for i := range byte(4) {
		putCharHi(win, first+i, 2)
	}
	setCursor(win, x, y+1)
	for i := range byte(4) {
		putCharHi(win, first+i+4, 2)
	}
}

// maze_main_loop (WMAZE 7020) reads input, moves and runs the cell events until the game
// state leaves 5.
func mazeMainLoop() {
	if cellEventIndex >= 0 && int16(mazeB.FirstEvent[partyBlock]) > cellEventIndex {
		cellEventIndex = int16(mazeB.FirstEvent[partyBlock])
	}
	if returnCmd != 23 {
		floodMarkCells(int(partyBlock), int(cellX), int(cellY), 0)
	}
	initMazeScreen()
	refreshAllWindows()
	returnCmd = 5
	gameState = 5
	mzG503e = 1
	for {
		if mzRedrawView == 1 {
			redrawMazeView()
		}
		if testBit(reachableCells[:], partyCell()) {
			startEncounter(1)
			return
		}
		if testBit(eventCells[:], partyCell()) && mzG5034 == 1 {
			runCellEvents()
			if gameState != 5 {
				return
			}
		}
		s := &saveState
		s.G4f80++
		if s.G4f80%10 == 5 {
			for m := 0; m < partyCount; m++ {
				pc := &party[m]
				if pc.Condition >= 3 {
					continue
				}
				spells := 0
				if int(s.G4f80%60/10) == m {
					spells = 1
					pc.Stamina -= int16(pc.LoadCat) + 1
					if pc.Stamina < 0 {
						pc.Stamina = 0
					}
				}
				healMember(m, 0, 0, 1, spells)
			}
		}
		well, asleep := 0, 0
		for m := 0; m < partyCount; m++ {
			if party[m].Condition == 0 {
				well++
			}
			if party[m].Status[2] > 0 || party[m].Status[3] > 0 {
				asleep++
			}
		}
		if well == 0 {
			gameState = 8
			return
		}
		if asleep == partyCount && restParty() {
			startEncounter(0)
			return
		}
		mzRedrawView = 0
		mzG5034 = 0
		if random(1000) >= 995 {
			startEncounter(0)
			return
		}
		if mzG503c == 1 {
			refreshAllWindows()
			mzG503c = 0
		}
		switch {
		case (gameStateArg == 5 || gameStateArg == 12 && partyBlock > 8) && testBit(mazeA.Bits1082[:], partyCell()):
			eraseWindow(mainWindow, true)
			delayLong()
			showMazeMessage(2023)
			mzG503e = 1
		case gameStateArg == 8 && partyCounters[0].B == 1:
			eraseWindow(mainWindow, true)
			delayLong()
			showMazeMessage(2054)
			mzG503e = 1
		case mzG503e == 1:
			eraseWindow(mainWindow, true)
			mzG503e = 0
		}
		setClickRegion(0, 33, 20, 34, 21)
		setClickRegion(1, 36, 22, 37, 23)
		setClickRegion(2, 32, 22, 35, 23)
		setClickRegion(3, 30, 22, 31, 23)
		setClickRegion(4, 21, 20, 27, 23)
		setClickRegion(5, 255, 0, 0, 0)
		switch in := getMazeInput(mzMsg2000); in {
		case 0:
			mzAnimCounter = 0
			moveParty(0)
			tickEffectTimers(1)
		case 1, 3:
			mzAnimCounter = 0
			if g4f88 == 0 {
				partyFacing = (partyFacing + int16(in)) % 4
				mzRedrawView = 1
				if partyFacing == g4f8a {
					mzG5034 = 1
					cellEventIndex = int16(mazeB.FirstEvent[partyBlock])
				}
			} else if in == 3 {
				moveParty(-1)
			} else {
				moveParty(1)
			}
			tickEffectTimers(1)
		case 2: // switch the arrow pad between turning and side-stepping
			g4f88 = (g4f88 + 1) % 2
			deferWindowDraw(g4fb0)
			if g4f88 == 0 {
				wmazeF6f30(g4fb0, 13, 1, 5, 6, 7, 8)
				wmazeF6f30(g4fb0, 10, 3, 1, 13, 3, 14)
				wmazeF6fa3(g4fb0, 12, 3, 17)
				wmazeF6f30(g4fb0, 16, 3, 15, 10, 16, 12)
			} else {
				wmazeF6f30(g4fb0, 13, 1, 5, 6, 7, 8)
				wmazeF6f30(g4fb0, 10, 3, 1, 2, 3, 4)
				wmazeF6fa3(g4fb0, 12, 3, 25)
				wmazeF6f30(g4fb0, 16, 3, 9, 10, 11, 12)
			}
			resumeWindowDraw(g4fb0)
			refreshWindow(g4fb0)
			mzRedrawView = 1
		case 4:
			mazePartyMenu()
		}
		if gameState != 5 {
			return
		}
	}
}

// set_map_offset (WMAZE 75d7) puts the party at map position (x, y).
func setMapOffset(x, y int16) {
	partyX, partyY = x, y
	moveToMazeCell(&cellX, &cellY, &partyBlock, int(partyX), int(partyY))
}

// can_enter_cell (WMAZE 7601) steps the party one cell in direction dir unless it may not
// go there; true if it stays (blocked). visited is the flight's cell set.
func canEnterCell(dir int, visited []byte) bool {
	c := partyCell()
	if f := getBitfield(mazeA.Floor[:], c, 4); (f == 1 || f == 2) && getBitfield(mazeA.Feature[:], c, 2) == dir {
		return true
	}
	x, y := partyX, partyY
	stepPosition(dir, 1)
	if moveToMazeCell(&cellX, &cellY, &partyBlock, int(partyX), int(partyY)) != 0 {
		setMapOffset(x, y)
		return true
	}
	c = partyCell()
	if testBit(visited, c) || testBit(reachableCells[:], c) {
		setMapOffset(x, y)
		return true
	}
	if testBit(mazeA.Bits1082[:], c) {
		switch gameStateArg {
		case 0, 1, 2, 3, 4, 6, 8, 10, 12:
			if gameStateArg != 12 || partyBlock < 9 {
				setMapOffset(x, y)
				return true
			}
		}
	}
	if testBit(mazeA.Bits1178[:], c) && gameStateArg == 2 {
		setMapOffset(x, y)
		return true
	}
	cellEventIndex = int16(mazeB.FirstEvent[partyBlock])
	return false
}

// try_step (WMAZE 77a8) steps the party forward if it can; true if it is blocked.
func tryStep(visited []byte) bool {
	if getWallInDirection(int(partyFacing), int(partyBlock), int(cellX), int(cellY), int(partyX), int(partyY)) >= 2 {
		return true
	}
	return canEnterCell(int(partyFacing), visited)
}

// show_prev_position (WMAZE 77ea) shows the view from the saved position for a moment.
func showPrevPosition() {
	x, y, cx, cy, b := partyX, partyY, cellX, cellY, partyBlock
	s := &saveState
	partyX, partyY, cellX, cellY, partyBlock = s.G4f96, s.G4f94, s.G4f92, s.G4f90, s.G4f8e
	redrawMazeView()
	waitAnimating(2)
	partyX, partyY, cellX, cellY, partyBlock = x, y, cx, cy, b
}

// show_fleeing_msg (WMAZE 7859).
func showFleeingMsg() bool {
	refreshAllWindows()
	showMazeMessage(2041)
	return true
}

// flee_party (WMAZE 7868) makes the party flee up to steps cells (game state 23).
func fleeParty(steps int) {
	var visited [96]byte
	returnCmd = 23
	gameState = 5
	eraseWindow(mainWindow, false)
	shown := false
	setBit(visited[:], partyCell(), true)
	away := partyFacing
	if int16(gameStateArg) == saveState.G4f98 {
		away = (saveState.G4f8c + 2) % 4
	}
	turn := int16(1)
	if (partyFacing+3)%4 == away {
		turn = 3
	}
	for partyFacing != away {
		partyFacing = (partyFacing + turn) % 4
		redrawMazeView()
		if !shown {
			shown = showFleeingMsg()
		}
		waitAnimating(2)
	}
	side := int16(-1)
	for {
		back := false
		savePosition()
		if side != -1 && random(4) == 0 {
			partyFacing = (partyFacing + side) % 4
			if tryStep(visited[:]) {
				partyFacing = (partyFacing + 2) % 4
				if tryStep(visited[:]) {
					partyFacing = saveState.G4f8c
				} else {
					if !shown {
						shown = showFleeingMsg()
					}
					showPrevPosition()
					back = true
				}
			} else {
				if !shown {
					shown = showFleeingMsg()
				}
				showPrevPosition()
				back = true
			}
		}
		if !back && tryStep(visited[:]) {
			t := int16(3)
			if random(2) == 0 {
				t = 1
			}
			partyFacing = (t + partyFacing) % 4
			if !tryStep(visited[:]) {
				showPrevPosition()
			} else {
				partyFacing = (partyFacing + 2) % 4
				if !tryStep(visited[:]) {
					showPrevPosition()
				} else {
					partyFacing = saveState.G4f8c
					eraseWindow(mainWindow, true)
					return
				}
			}
		}
		setBit(visited[:], partyCell(), true)
		redrawMazeView()
		if !shown {
			shown = showFleeingMsg()
		}
		waitAnimating(2)
		side = 3
		if random(2) != 0 {
			side = 1
		}
		steps--
		if steps <= 0 || wmazeF7bd2(int(cellX), int(cellY), int(partyBlock), int(cellEventIndex)) != -1 {
			break
		}
	}
	tickEffectTimers(2)
	waitAnimating(messageDelay / 4)
	eraseWindow(mainWindow, true)
}
