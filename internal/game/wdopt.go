package game

import (
	"github.com/retro-ware/wiz6-go/internal/drv"
)

// WDOPT: casting a spell (game state 19) and using an item (20) outside combat, for
// cur_member; then back to the maze. The maze view copies of the overlay (8096..9e83),
// load_sound (45ac), the idle step (4688), dice (4c76, 4fe2), the member name (489b) and
// the event helpers (5011, 5051, 53c4) are WMAZE's, the spell window pieces WPOPS's.

// WDOPT bss (0x502a..0x55fa). The blinking cursor (g502e, g5030..g5053) is WPCVW's
// pvBlink* state: the two overlays never run together.
var (
	dpResultMsg  int    // g502a: the message to show when done (0 none)
	dpResultBeep int    // g502c: beep with it
	dpAnimCount  int    // g506a: counts to 10 between animation frames
	dpKeys       string // 0x50bc: message 2001, the menu keys
	dpTarget     int    // g51b6: the member a spell is cast on

	// The overhead map (show_overhead_map): glyphs by [col][row] of the floor (g50c6),
	// the north wall (g50fe), the west wall (g5132) and the corner (g517a).
	dpMapFloor, dpMapNorth, dpMapWest, dpMapCorner [5][5]int
	dpMapX, dpMapY                                 int // g51ac, g51ae: the map's text column and row
	// The map's glyphs, set by the spell's power (g50f8 .. g51b4).
	dpGlDark, dpGlNoNorth, dpGlNoWest, dpGlSolid int // g50f8, g50fc, g5130, pclist_sex
	dpGlStairUp, dpGlStairDown, dpGlVoid         int // g5164, g5166, g5168
	dpGlSpecial                                  int // g516a
	dpGlN1, dpGlN3, dpGlNDoor                    int // g516c, g516e, g5170
	dpGlW1, dpGlW3, dpGlWDoor                    int // g5172, g5174, g5176
	dpGlCorner, dpGlFloor, dpGlBorder, dpGlParty int // g5178, g51b2, g51b0, g51b4

	// The use item list (do_use_item): the slots shown (g51ba, g51b8 of them), and by
	// slot the item's name (0x51e6) and quantity (g5346, 0 none).
	dpItemSlots []int
	dpItemNames [22]string
	dpItemQty   [22]int
)

// ovmain (WDOPT 4572).
func wdoptOvmain() {
	dpResultMsg, dpResultBeep, pvBlinkState = 0, 0, -1
	if gameState == 19 {
		castSpellInMaze(int(curMember))
	}
	if gameState == 20 {
		useItemInMaze(int(curMember))
	}
	gameState = 5
}

// viewAnimStep is the idle step of WDOPT's waits: every 10th call flips the animation
// phase and redraws the view.
func viewAnimStep() {
	if dpAnimCount == 0 {
		animPace(mazeAnimPeriod)
		animPhase = (animPhase + 1) % 2
		redrawMazeView()
	} else {
		idleStep()
	}
	dpAnimCount = (dpAnimCount + 1) % 10
}

// get_menu_choice_blink (WDOPT 4728) waits for a key of keys (its index) or a click
// region (its index, menuClicked set), blinking the cursor and animating the view.
func getMenuChoiceView(keys string) int {
	phase := animPhase
	if pvBlinkState != -1 {
		blinkCursor()
	}
	menuClicked = 0
	for {
		if keyAvailable(1) {
			if c := charIndex(keys, int(uint8(getKey(1)))); c >= 0 {
				return c
			}
			continue
		}
		r := checkClickRegions(clickRegions[:])
		if clickFlag {
			menuClicked = 1
			return r - 1
		}
		if dpAnimCount == 0 {
			animPace(mazeAnimPeriod)
			animPhase = (animPhase + 1) % 2
			redrawMazeView()
			if pvBlinkState != -1 && animPhase == phase {
				blinkCursor()
			}
		} else {
			idleStep()
		}
		dpAnimCount = (dpAnimCount + 1) % 10
	}
}

// wait_animated_dopt (WDOPT 47ce) animates the view for ticks; a click or Enter ends it.
func waitAnimatedView(ticks int) {
	flushKeys()
	for ticks *= 5; ticks > 0; ticks-- {
		if checkScreenClick() {
			return
		}
		if keyAvailable(1) && int8(getKey(1)) == 13 {
			return
		}
		viewAnimStep()
	}
}

// choose_target_member (WDOPT 48e0) asks with message msg for the member to target;
// the member or -1 (0 with fewer than two).
func wdoptChooseTarget(msg int) int {
	if partyCount < 2 {
		return 0
	}
	win := createWindow(0, 19, 19, 5, 25, -4, 0, false)
	fillWindow(win, ' ', 3)
	cancelText, cancelHi := getMessage(2027), getMessage(2028)
	title := getMessage(msg)
	showTitleText(title, 10-(len(title)+6)/2)
	cx, cy := int(mainWindow.CurX), int(mainWindow.CurY)
	refreshWindow(mainWindow)
	refreshWindow(win)
	setCursor(mainWindow, cx, cy)
	if mousePresent {
		printStringHi(mainWindow, cancelText, 3)
	} else {
		printString(mainWindow, cancelHi, -5)
	}
	for m := 0; m < partyCount; m++ {
		x, y := memberCell(m)
		setCursor(win, x, y)
		printStringHi(win, memberName(m), 3)
		setClickRegion(m, win.X+byte(x), win.Y+byte(y), byte(len(memberName(m))+int(win.X)+x-1), win.Y+byte(y))
	}
	setClickRegion(partyCount, mainWindow.X+byte(cx), mainWindow.Y+byte(cy), byte(len(cancelHi)+int(mainWindow.X)+cx-1), mainWindow.Y+byte(cy))
	setClickRegion(partyCount+1, 255, 0, 0, 0)
	sel := -1
	for {
		if !mousePresent {
			if sel == -1 {
				setCursor(mainWindow, cx, cy)
				printString(mainWindow, cancelHi, -5)
			} else {
				x, y := memberCell(sel)
				setCursor(win, x, y)
				printString(win, memberName(sel), -5)
			}
		}
		c := getMenuChoiceView(dpKeys)
		if menuClicked == 1 {
			sel = -1
			if c < partyCount {
				sel = c
			}
			break
		}
		switch c {
		case 1:
			if sel != -1 && sel%2 > 0 {
				drawMemberName(win, sel)
				sel--
			}
		case 2:
			if sel != -1 {
				drawMemberName(win, sel)
				if sel -= 2; sel < 0 {
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
				setCursor(mainWindow, cx, cy)
				printStringHi(mainWindow, cancelText, 3)
				sel = 0
			} else if sel+2 < partyCount {
				drawMemberName(win, sel)
				sel += 2
			}
		}
		if c == 5 {
			break
		}
	}
	destroyWindow(win, false)
	return sel
}

// copy_name (WDOPT 4ca5, WPOPS 6fea) is name in lower case.
func copyName(name string) string {
	b := []byte(name)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

// removeItem (remove_item, WDOPT 4e09, WMNPC) removes item slot of member m: an inventory item (slot < 10)
// is unequipped first; swag slots are 10 on.
func removeItem(m, slot int) {
	p := &party[m]
	if slot >= 10 {
		p.SwagCount--
		for i := slot; i < int(p.SwagCount)+10; i++ {
			p.Items[i] = p.Items[i+1]
		}
		return
	}
	for s := 0; s < 8; s++ {
		if int(p.Equipped[s]) != slot {
			continue
		}
		switch s {
		case 0, 1:
			clearHandAttack(m, s)
		case 2:
			p.AC[2] = 0
		default:
			p.AC[s] = p.AC[0]
		}
		p.Equipped[s] = 255
	}
	p.Items[slot].ID = 0
	p.ItemCount--
	for i := slot; i < int(p.ItemCount); i++ {
		p.Items[i] = p.Items[i+1]
		for s := 0; s < 8; s++ {
			if int(p.Equipped[s]) == i+1 {
				p.Equipped[s] = uint8(i)
			}
		}
	}
}

// useItemCharge (use_item_charge, WDOPT 4faf, WMNPC) uses a charge of item slot of member m, removing the item
// with its last.
func useItemCharge(m, slot int) {
	p := &party[m]
	it := p.Items[slot]
	if it.Qty > 0 {
		it.Qty--
		p.Items[slot] = it
		if it.Qty == 0 {
			removeItem(m, slot)
		}
	}
}

// show_result_message (WDOPT 5082) shows message msg on the message line for a moment,
// with a beep if dpResultBeep.
func wdoptShowResult(msg int) {
	eraseWindow(mainWindow, false)
	refreshAllWindows()
	printCentered(mainWindow, getMessage(msg), 0, 18)
	if dpResultBeep != 0 {
		playSound(0)
	}
	waitAnimatedView(messageDelay / 2)
	eraseWindow(mainWindow, false)
}

// isLockAhead is whether event ev is a lock (kind 7) on the party's facing.
func isLockAhead(ev int) bool {
	b := extractBits(int(uint16(mazeB.DirBits[ev])), int(partyFacing)<<1, 2)
	return mazeB.Kind[ev] == 7 && (b == 1 || b == 2)
}

// find_lock_event (WDOPT 50e8) is the lock event on the party's cell and facing, or -1.
func findLockEvent() int {
	for ev := int(mazeB.FirstEvent[partyBlock]); ; ev++ {
		if ev = findCellEvent(int(cellX), int(cellY), int(partyBlock), ev); ev == -1 {
			return -1
		}
		if isLockAhead(ev) {
			return ev
		}
	}
}

// try_open_lock (WDOPT 517b) is the unlocking spell of the given power on the lock
// ahead: it opens, resists (1) or is jammed (2), message 2113+result; -1 with no lock.
func tryOpenLock(power int) int {
	ev := findLockEvent()
	if ev == -1 {
		return -1
	}
	level := extractBits(int(mazeB.Param[ev]), 0, 5)
	jammed := extractBits(int(uint16(mazeB.DirBits[ev])), int(partyFacing)<<1, 2) == 2 && level == 0
	result := 0
	if int(int16(random(power<<1)+random(power<<1)+random(power<<1)+3))/2 < level { // C int sum wraps
		result = 1
	}
	if jammed {
		result = 2
	}
	if result == 0 {
		openLockAhead(ev)
	}
	dpResultMsg, dpResultBeep = result+2113, result
	return 0
}

// mark_map_neighbor (WDOPT 5357) sets the floor glyph of the map cell next to (col, row)
// in direction dir.
func wdoptMarkNeighbor(dir, v, col, row int) {
	switch dir {
	case 0:
		row--
	case 1:
		col++
	case 2:
		row++
	case 3:
		col--
	}
	if col >= 0 && col < 5 && row >= 0 && row < 5 {
		dpMapFloor[col][row] = v
	}
}

// map_cell (WDOPT 546a) fills map cell (col, row) from map position (x, y).
func wdoptMapCell(x, y, col, row int) {
	cx, cy, blk := cellX, cellY, partyBlock
	if moveToMazeCell(&cx, &cy, &blk, x, y) != 0 {
		if gameStateArg != 10 && gameStateArg != 12 {
			dpMapFloor[col][row] = dpGlSolid
			dpMapNorth[col][row] = dpGlNoNorth
			dpMapWest[col][row] = dpGlNoWest
			return
		}
		dpMapFloor[col][row] = dpGlDark
		return
	}
	cell := cellIndex(int(blk), int(cx), int(cy))
	floor := getBitfield(mazeA.Floor[:], cell, 4)
	feat := getBitfield(mazeA.Feature[:], cell, 2)
	g := -1
	switch floor {
	case 1:
		wdoptMarkNeighbor(feat, dpGlStairUp, col, row)
	case 2:
		wdoptMarkNeighbor(feat, dpGlStairDown, col, row)
	case 13:
		g = dpGlDark
	case 14:
		g = dpGlVoid
	}
	special := testBit(mazeA.SpecialCells[:], cell)
	if special {
		g = dpGlSpecial
	}
	if testBit(mazeA.SolidCells[:], cell) {
		g = dpGlSolid // the C's following switch on the map changes nothing
	}
	if special {
		switch gameStateArg {
		case 0, 4, 5, 12:
			if gameStateArg == 12 && blk < 9 {
				g = dpGlDark
			} else {
				g = dpGlSpecial
			}
		case 1, 2, 3, 6, 11:
			g = dpGlVoid
		case 8, 10:
			g = dpGlDark
		}
	}
	if g != -1 {
		dpMapFloor[col][row] = g
	}
	g = -1
	switch getWallInDirection(0, int(blk), int(cx), int(cy), x, y) {
	case 1:
		g = dpGlN1
	case 2:
		g = dpGlNoNorth
	case 3:
		g = dpGlN3
	}
	if floor == 7 && feat == 0 {
		g = dpGlNDoor
	}
	if g != -1 {
		dpMapNorth[col][row] = g
	}
	g = -1
	switch getWallInDirection(3, int(blk), int(cx), int(cy), x, y) {
	case 1:
		g = dpGlW1
	case 2:
		g = dpGlNoWest
	case 3:
		g = dpGlW3
	}
	if floor == 7 && feat == 3 {
		g = dpGlWDoor
	}
	if g != -1 {
		dpMapWest[col][row] = g
	}
}

// set_map_corner (WDOPT 581d) picks the corner glyph of map cell (col, row) from the
// walls that meet there.
func wdoptMapCorner(col, row int) {
	g, west, north := -1, 1, 2
	if dpMapNorth[col][row] != -1 {
		g = dpGlCorner
	}
	if dpMapWest[col][row] != -1 {
		g, north = dpGlCorner, 0
	}
	if col-1 >= 0 && dpMapNorth[col-1][row] != -1 {
		g, west = dpGlCorner, 0
	}
	if row-1 >= 0 && dpMapWest[col][row-1] != -1 {
		g = dpGlCorner
	}
	if g != -1 {
		dpMapCorner[col][row] = g + west + north
	}
}

// wdoptMapGlyph draws glyph g of picture slot 4 at pixel (x, y), clipped to lo..hi.
func wdoptMapGlyph(x, y, lo, hi, g int) {
	drv.DrawFrames(4, x, y, lo, hi, 0, false, []byte{byte(g)})
}

// draw_map_party (WDOPT 58f8) draws the members' markers in map cell (col, row), two to
// a glyph, left and right.
func wdoptMapParty(col, row, glyph int) {
	x, y := dpMapX<<3+col<<4, dpMapY<<3+row<<4
	for i := 0; i < partyCount; i++ {
		wdoptMapGlyph(x, y, x, x+16-(i+1)%2*6, i/2+glyph)
	}
}

// draw_map_floor (WDOPT 59c4).
func wdoptMapFloorGlyph(col, row, g int) {
	wdoptMapGlyph(dpMapX<<3+col<<4, dpMapY<<3+row<<4, dpMapX<<3+1, 239, g)
}

// draw_map_walls (WDOPT 5a2f) draws the north wall, west wall and corner glyphs of map
// cell (col, row) (-1 none).
func wdoptMapWalls(col, row, north, west, corner int) {
	x, y, lo := dpMapX<<3+col<<4, dpMapY<<3+row<<4, dpMapX<<3+1
	if north != -1 {
		wdoptMapGlyph(x+3, y, lo, 239, north)
	}
	if west != -1 {
		wdoptMapGlyph(x, y+2, lo, 239, west)
	}
	if corner != -1 {
		wdoptMapGlyph(x, y, lo, 239, corner)
	}
}

// draw_map_border (WDOPT 5b4f) draws the map's top and bottom edges.
func wdoptMapBorder(cols, rows int) {
	for i := 0; i < cols<<1; i++ {
		wdoptMapGlyph((dpMapX+i)<<3, dpMapY<<3-1, 72, 248, dpGlBorder)
		wdoptMapGlyph((dpMapX+i)<<3, (rows<<1+dpMapY)<<3, 72, 248, dpGlBorder)
	}
}

// show_overhead_map (WDOPT 5be2) shows the map around the party over the view, 3x3 or
// (power above 3) 5x5 cells, with more detail at higher power, until Enter or a click.
func wdoptShowOverheadMap(power int) {
	if dtLog != nil { // stubbed in the difftest
		return
	}
	cols, center := 3, 1
	if power > 3 {
		cols, center = 5, 2
	}
	rows := cols
	dpMapX, dpMapY = 30-cols<<1, 5
	low := power == 1 || power == 4
	mid := power == 2 || power == 5
	pick := func(m, h int) int { // -1 at low power, m at middle, h at high
		switch {
		case low:
			return -1
		case mid:
			return m
		}
		return h
	}
	dpGlCorner, dpGlNoNorth, dpGlNoWest = pick(1, 1), pick(5, 5), pick(9, 9)
	dpGlN1, dpGlN3, dpGlNDoor = pick(5, 6), pick(5, 7), pick(5, 8)
	dpGlW1, dpGlW3, dpGlWDoor = pick(9, 10), pick(9, 11), pick(9, 12)
	dpGlFloor, dpGlSolid, dpGlDark, dpGlVoid = 13, 14, 15, 16
	dpGlStairUp, dpGlStairDown = 13, 13
	if power == 3 || power == 6 {
		dpGlStairUp, dpGlStairDown = 17, 18
	}
	dpGlBorder, dpGlSpecial, dpGlParty = 19, dpGlVoid, 20
	for c := range 5 {
		for r := range 5 {
			dpMapFloor[c][r], dpMapNorth[c][r], dpMapWest[c][r], dpMapCorner[c][r] = dpGlFloor, -1, -1, -1
		}
	}
	loadMonsterPicSlot(0, 4)
	win := createWindow(dpMapX, dpMapY, cols<<1, rows<<1, 20, -1, 2, false)
	clearClickFlag()
	flushKeys()
	for waiting := true; waiting; {
		animPace(mazeAnimPeriod)
		animPhase = (animPhase + 1) % 2
		drawMazeView()
		drvFillBack(win, 0)
		wdoptMapBorder(cols, rows)
		for r := 0; r < rows; r++ {
			for c := 0; c < cols; c++ {
				wdoptMapCell(int(partyX)+c-center, int(partyY)-r+center, c, r)
			}
		}
		for r := 0; r < rows; r++ {
			for c := 0; c < cols; c++ {
				wdoptMapCorner(c, r)
			}
		}
		for r := 0; r < rows; r++ {
			for c := 0; c < cols; c++ {
				wdoptMapFloorGlyph(c, r, dpMapFloor[c][r])
			}
		}
		for r := 0; r < rows; r++ {
			for c := 0; c < cols; c++ {
				wdoptMapWalls(c, r, dpMapNorth[c][r], dpMapWest[c][r], dpMapCorner[c][r])
			}
		}
		wdoptMapParty(center, center, dpGlParty)
		drvShowWindow(pictureWin)
		if checkScreenClick() {
			waiting = false
		}
		if keyAvailable(1) && int8(getKey(1)) == 13 {
			waiting = false
		}
		// ponytail: the C loops at machine speed; a tick per frame stands in for the
		// 1990 PC's draw time so the animation does not strobe.
		delayTick()
	}
	destroyWindow(win, false)
	redrawMazeView()
	dpAnimCount = 0
	picSlotMonster[4+0], picSlotLoaded[4+0] = -1, false
}

// f6054 (WDOPT) applies spell s of the given power cast (or used from an item) by member
// caster on target outside combat. Spells with no effect here do nothing.
func applySpellOutsideCombat(caster, s, power, target int) {
	t := &party[target]
	pc := &partyCounters
	sub := func(v uint8, n int) uint8 { // v - n, at least 0
		return uint8(max(int(int16(int(v)-n)), 0)) // C int wraps
	}
	switch s {
	case 53:
		pc[2].A += int16(power * 50)
		pc[2].B = int16(clampToward(power*5+70, int(pc[2].B), 0))
	case 68:
		pc[3].A += int16(power * 35)
		pc[3].B = int16(clampToward((power-1)/2+1, int(pc[3].B), 0))
	case 41:
		pc[1].A += int16(power * 50)
		pc[1].B = int16(clampToward((power-1)/2+1, int(pc[1].B), 0))
	case 30:
		pc[5].A += int16(power * 5)
	case 38:
		pc[0].A += int16(100 * power)
	case 70:
		pc[4].A += int16(power * 50)
		pc[4].B = int16(clampToward(power*5, int(pc[4].B), 0))
	case 10:
		t.Stamina = min(t.Stamina+int16(rollDice(formatDice(power<<1, 8, power))), t.StaminaMax)
	case 15:
		t.Status[3] = sub(t.Status[3], power*25)
	case 26:
		if t.PoisonStr = sub(t.PoisonStr, power); t.PoisonStr == 0 {
			t.Status[5] = 0
		}
		if t.Status[5] = sub(t.Status[5], power*5); t.Status[5] == 0 {
			t.PoisonStr = 0
		}
	case 46:
		if t.Condition == 2 && int(int16(power*3)) > 21-int(t.Stats[3]) { // C int wraps
			t.Stats[3]--
			if random(21) <= int(t.Stats[3]) {
				t.Condition = 0
			} else {
				t.Condition, t.HP, t.Stamina = 3, 0, 0
			}
		}
	case 57:
		t.Status[7] = sub(t.Status[7], power*25)
	case 51:
		for _, i := range []int{1, 2, 4, 8, 9} {
			t.Status[i] = sub(t.Status[i], power<<1)
		}
	case 64:
		t.HP = min(t.HP+int16(rollDice(formatDice(power, 7, power))), t.HPMax)
	case 77:
		if t.Condition == 3 && int(int16(power*3)) > 18-int(t.Stats[3]) { // C int wraps
			t.Stats[3]--
			t.Condition, t.HP, t.Stamina = 0, 1, t.StaminaMax/10
			t.Unk026++
		}
	case 39:
		tryOpenLock(power)
	case 60:
		wdoptShowOverheadMap(power)
	case 80:
		c := &party[caster]
		c.Stamina = min(c.Stamina+int16(rollDice(formatDice(power, 8, power))), c.StaminaMax)
		c.HP = min(c.HP+int16(rollDice(formatDice(power, 2, 0))), c.HPMax)
	case 81:
		c := &party[caster]
		for r := range 6 {
			c.SP[r].Cur = min(c.SP[r].Cur+int16(rollDice(formatDice(power, 10, 0))), c.SP[r].Max)
		}
	}
}

// check_item_location_event (WDOPT 6561) is whether item slot of member m does
// something on the party's cell: an item event (kind 19) for it swaps its flags and plays
// its sound; the key of the lock ahead opens it.
func wdoptItemLocationEvent(m, slot int) bool {
	id := party[m].Items[slot].ID
	for ev := int(mazeB.FirstEvent[partyBlock]); ; ev++ {
		if ev = findCellEvent(int(cellX), int(cellY), int(partyBlock), ev); ev == -1 {
			return false
		}
		bits := int(uint16(mazeB.DirBits[ev]))
		if mazeB.Kind[ev] == 19 && id == mazeB.Data[ev] {
			setBit(levelFlags[gameStateArg][:], extractBits(bits, 0, 8), false)
			if b := extractBits(bits, 8, 8); b != 0 {
				setBit(levelFlags[gameStateArg][:], b, true)
			}
			if snd := int(mazeB.Param[ev]); snd >= 14 {
				if snd != curSound {
					loadSoundBank(snd)
				}
				playSound(14)
			} else if snd > 0 {
				playSound(snd)
			}
			cellEventIndex = int16(ev + 1)
			return true
		}
		if isLockAhead(ev) && id == mazeB.Unk288[ev] {
			openLockAhead(ev)
			dpResultMsg, dpResultBeep = 2113, 0
			return true
		}
	}
}

// draw_item_list (WDOPT 76d5) shows five rows of the item list around index sel, the one
// at sel highlighted when hi.
func drawSlotItemList(win *Window, m, sel int, hi bool) {
	drawListFrame(win, false)
	deferWindowDraw(win)
	for row := -2; row <= 2; row++ {
		if sel+row < 0 || sel+row >= len(dpItemSlots) {
			continue
		}
		setCursor(win, 2, row+2)
		slot := dpItemSlots[sel+row]
		if hi && row == 0 {
			printString(win, dpItemNames[slot], 3)
			if dpItemQty[slot] > 0 {
				for win.CurX < 15 {
					putChar(win, ' ', 3)
				}
				printInt(win, dpItemQty[slot], 3, 1)
			} else {
				for win.CurX < 18 {
					putChar(win, ' ', 3)
				}
			}
		} else {
			printStringHi(win, dpItemNames[slot], 3)
			if dpItemQty[slot] > 0 {
				setCursor(win, 15, row+2)
				printInt(win, dpItemQty[slot], 3, 18)
			}
		}
		setCursor(win, 18, row+2)
		putCharHi(win, party[m].Items[slot].Icon+1, 4)
	}
	resumeWindowDraw(win)
	refreshWindow(win)
}

// do_use_item (WDOPT 7860): member m picks an item and uses it: an item event or lock
// here, the spell it holds, a flag-swapping item of this map (kind 4) or the map 8 item
// (kind 5); otherwise "nothing happens".
func useItemInMaze(m int) {
	dpKeys = getMessage(2001)
	dpAnimCount = 0
	deferWindowDraw(g4fb0)
	win := createWindow(0, 19, 19, 5, 10, -4, 0, false)
	fillWindow(win, ' ', 3)
	list := createWindow(20, 19, 20, 5, 20, -5, 0, false)
	fillWindow(list, ' ', 3)
	title := replaceCharWithString(getMessage(2033), copyName(memberName(m)), '$')
	showTitleText(title, 10-(len(title)+1)/2)
	eraseCursorMark(bottomLineWin, false)
	deferWindowDraw(win)
	setCursor(win, 2, 1)
	printMessage(2034, win, 18)
	resumeWindowDraw(win)
	drawListFrame(list, false)
	refreshAllWindows()
	p := &party[m]
	dpItemSlots = dpItemSlots[:0]
	add := func(slot int) {
		it := p.Items[slot]
		rec := itemRec(int(it.ID))
		dpItemNames[slot] = cstr(rec.Name[:])
		dpItemQty[slot] = 0
		if stackable[it.Type] == 1 {
			dpItemQty[slot] = int(it.Qty)
		}
		dpItemSlots = append(dpItemSlots, slot)
	}
	for i := 0; i < int(p.ItemCount); i++ {
		add(i)
	}
	for i := 0; i < int(p.SwagCount); i++ {
		add(i + 10)
	}
	cancel := getMessage(2035)
	sel, onCancel := 0, false
	setClickRegion(0, list.X+2, list.Y+2, list.X+19, list.Y+2)
	setClickRegion(1, list.X+1, list.Y, list.X+1, list.Y)
	setClickRegion(2, list.X+1, list.Y+4, list.X+1, list.Y+4)
	setClickRegion(3, win.X+7, win.Y+3, byte(len(cancel)+int(win.X)+6), win.Y+3)
	setClickRegion(4, 255, 0, 0, 0)
	for {
		setCursor(win, 7, 3)
		if onCancel {
			printString(win, cancel, -3)
		} else {
			printStringHi(win, cancel, 3)
		}
		drawSlotItemList(list, m, sel, !onCancel)
		c := getMenuChoiceView(dpKeys)
		if menuClicked == 1 {
			switch c {
			case 0:
				c = 5
			case 1:
				c = 2
			case 2:
				c = 4
			case 3:
				onCancel, c = true, 5
			}
		}
		switch c {
		case 1:
			onCancel = true
		case 2:
			if onCancel {
				onCancel = false
			} else if sel > 0 {
				sel--
			}
		case 3:
			onCancel = false
		case 4:
			if onCancel {
				onCancel = false
			} else if sel+1 < len(dpItemSlots) {
				sel++
			}
		}
		if c == 5 {
			break
		}
	}
	slot := 0 // the C reads a stale g51ba[0] with no items
	if len(dpItemSlots) > 0 {
		slot = dpItemSlots[sel]
	}
	if !onCancel {
		if wdoptItemLocationEvent(m, slot) {
			useItemCharge(m, slot)
		} else {
			rec := itemRec(int(p.Items[slot].ID))
			switch {
			case rec.Spell != 0:
				level, spell := int(rec.Spell)/256, int(rec.Spell)%256
				if spellTable[spell].Target == 1 {
					refreshAllWindows()
					if dpTarget = wdoptChooseTarget(2032); dpTarget == -1 {
						spell = -1
					}
				}
				if spell != -1 {
					useItemCharge(m, slot)
					applySpellOutsideCombat(m, spell, level, dpTarget)
				}
			case rec.Unk69 == 4 && gameStateArg == int(rec.Damage.Base):
				// A flag item of this map: clear bit Count, set bit Sides.
				mp := int(rec.Damage.Base)
				setBit(levelFlags[mp][:], int(rec.Damage.Count), false)
				setBit(scenarioHdr.LevelFlags[mp][:], int(rec.Damage.Count), false)
				setBit(levelFlags[mp][:], int(rec.Damage.Sides), true)
				setBit(scenarioHdr.LevelFlags[mp][:], int(rec.Damage.Sides), true)
				useItemCharge(m, slot)
			case rec.Unk69 == 5:
				if gameStateArg == 8 {
					partyCounters[0].B = 1
				} else {
					dpResultMsg, dpResultBeep = 2055, 1
				}
			default:
				dpResultMsg, dpResultBeep = 2046, 1
			}
		}
	}
	destroyWindow(win, false)
	destroyWindow(list, false)
	resumeWindowDraw(g4fb0)
	eraseWindow(mainWindow, false)
	eraseCursorMark(bottomLineWin, false)
	if dpResultMsg > 0 {
		wdoptShowResult(dpResultMsg)
	}
}

// do_cast_spell (WDOPT 7f30): member m picks a spell and power, pays the spell points
// and stamina (passing out at 0) and casts it.
func castSpellInMaze(m int) {
	dpKeys = getMessage(2001)
	dpAnimCount = 0
	deferWindowDraw(g4fb0)
	win := createWindow(0, 19, 19, 5, 10, -4, 0, false)
	fillWindow(win, ' ', 3)
	s := chooseSpellBook(m, func() int { return getMenuChoiceView(dpKeys) }, func(msg int) int { dpTarget = wdoptChooseTarget(msg); return dpTarget })
	pvBlinkState = -1
	if s != -1 {
		level, spell := s/256, s%256
		p := &party[m]
		sp := &spellTable[spell]
		p.SP[sp.Realm].Cur -= int16(int(sp.Cost) * level)
		applySpellOutsideCombat(m, spell, level, dpTarget)
		if p.Stamina -= int16(random(level) + int(sp.Level) + level); p.Stamina < 0 {
			p.Stamina = 0
			p.Status[2] = uint8(random(6) + 6)
		}
	}
	destroyWindow(win, false)
	resumeWindowDraw(g4fb0)
	eraseWindow(mainWindow, false)
	eraseCursorMark(bottomLineWin, false)
	if dpResultMsg > 0 {
		wdoptShowResult(dpResultMsg)
	}
}
