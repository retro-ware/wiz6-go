package game

// WPCMK: the character menu (create, review, delete, rename, change portrait) and the
// creation screens.

// WPCMK bss.
var (
	newPC       PC      // ds:5470: the character being made or shown (char_*, g5470..g561f)
	sheetWin    *Window // sheet_win: the character sheet, rows 0..19
	promptWin   *Window // prompt_win: the prompt below it, rows 20..24
	menuWin     *Window // menu_win: the list at the sheet's right
	bonusPoints int     // bonus_points: -1 hides them
	maxBonus    bool    // g56ce: 21 bonus points (never set in the shipped game)

	classEligible [14]bool // class_eligible

	// The roster list selectRosterCharacter shows (g5750 sex, g5770 race, g5790 class,
	// g57b0 slot, names at ds:56d0).
	rosterList []pclistEntry
)

// wpcmkMain (WPCMK ovmain) runs the character menu and goes back to the camp.
func wpcmkMain() {
	characterMenu()
	gameState = 4
}

// clearStatusLine (WPCMK 4b6d) blanks the status line at the sheet's top right.
func clearStatusLine() {
	setCursor(sheetWin, 21, 5)
	for {
		putCharHi(sheetWin, 32, 3)
		if sheetWin.CurX >= 39 {
			break
		}
	}
}

// printStatusCentered (WPCMK 4b9c) prints message msg centred on the status line.
func printStatusCentered(msg int) {
	clearStatusLine()
	t := getMessage(msg)
	setCursor(sheetWin, 30-(len(t)+1)/2, 5)
	printStringHi(sheetWin, t, 3)
}

// setFrameGlyph (WPCMK 4bed, WPCVW 5f86, WMAZE ceb2) sets the character of cell (x, y) of
// win without drawing it.
func setFrameGlyph(win *Window, x, y int, ch byte) {
	win.Cells[int(win.W)*y+x].Ch = ch
}

// drawCreationFrame (WPCMK 4c13) puts the character sheet's frame into win.
func drawCreationFrame(w *Window) {
	g := func(x, y int, ch byte) { setFrameGlyph(w, x, y, ch) }
	g(0, 0, 1)
	g(39, 0, 3)
	g(0, 19, 6)
	for i := 1; i < 39; i++ {
		g(i, 0, 2)
	}
	for i := 1; i < 20; i++ {
		g(i, 19, 7)
	}
	for i := 1; i < 19; i++ {
		g(0, i, 4)
	}
	for i := 1; i < 6; i++ {
		g(39, i, 5)
	}
	g(0, 4, 9)
	g(0, 13, 9)
	g(0, 15, 9)
	g(0, 17, 9)
	g(39, 4, 10)
	g(2, 19, 11)
	g(10, 19, 11)
	g(12, 19, 11)
	g(20, 19, 8)
	for i := 1; i < 39; i++ {
		g(i, 4, 12)
	}
	for i := 21; i < 39; i++ {
		g(i, 6, 7)
	}
	g(20, 4, 14)
	g(20, 5, 13)
	for i := 7; i < 19; i++ {
		g(20, i, 5)
	}
	g(20, 6, 21)
	g(20, 13, 10)
	g(20, 15, 10)
	g(20, 17, 10)
	g(39, 6, 8)
	for i := 1; i < 20; i++ {
		g(i, 13, 12)
		g(i, 15, 12)
		g(i, 17, 12)
	}
	for _, x := range []int{2, 10, 12} {
		g(x, 13, 14)
	}
	for i := 14; i < 19; i++ {
		g(2, i, 13)
		g(10, i, 13)
		g(12, i, 13)
	}
	for _, x := range []int{2, 10, 12} {
		g(x, 15, 16)
		g(x, 17, 16)
	}
	clearStatusLine()
}

// printStrAt (WPCMK 4ffd) prints s at (x, y).
func printStrAt(win *Window, x, y int, s string, color int) {
	setCursor(win, x, y)
	printString(win, s, color)
}

// printMsgAt (WPCMK 5022, WPCVW 50df) prints message msg at (x, y).
func printMsgAt(msg int, win *Window, x, y, color int) {
	t := getMessage(msg)
	setCursor(win, x, y)
	printString(win, t, color)
}

// printMsgColor (WPCMK 5058) prints message msg at the cursor.
func printMsgColor(msg int, win *Window, color int) {
	printString(win, getMessage(msg), color)
}

// putGlyphAt (WPCMK 507f, WPCVW 513c) draws glyph ch of font 2 at (x, y).
func putGlyphAt(win *Window, x, y int, ch byte) {
	setCursor(win, x, y)
	putCharHi(win, ch, 2)
}

// putIconAt (WPCMK 50a8, WPCVW 5165) draws glyph ch of font 4 at (x, y).
func putIconAt(win *Window, x, y int, ch byte) {
	setCursor(win, x, y)
	putCharHi(win, ch, 4)
}

// draw_portrait (WPCMK 5176, WPCVW a8ab) draws the 3x3 glyphs of portrait slot slot
// (font 2, from glyph 72) at (x, y). WPCMK always draws slot 0.
func drawPortrait(win *Window, slot, x, y int) {
	for r := range 3 {
		setCursor(win, x, y+r)
		for c := range 3 {
			putCharHi(win, byte(72+slot*9+r*3+c), 2)
		}
	}
}

// clearNameRows (WPCMK 5277) blanks the portrait's three rows.
func clearNameRows() {
	for i := range 3 {
		setCursor(sheetWin, 1, i+1)
		printStringHi(sheetWin, "   ", 3)
	}
}

// drawSexRaceClass (WPCMK 52b6) draws the name, sex, race and class.
func drawSexRaceClass() {
	printStrAt(sheetWin, 4, 1, cstr(newPC.Name[:]), 5)
	setCursor(sheetWin, 13, 1)
	putChar(sheetWin, getMessage(int(newPC.Sex) + 140)[0], 1)
	putChar(sheetWin, 45, 1)
	printMsgColor(int(newPC.Race)+100, sheetWin, 1)
	printMsgAt(int(newPC.Class)+120, sheetWin, 13, 2, 1)
}

// drawCharacterSheet (WPCMK 535b) draws the name, age, level, rank, experience, kills and
// the spell points.
func drawCharacterSheet() {
	w := sheetWin
	printStrAt(w, 4, 1, cstr(newPC.Name[:]), 5)
	setCursor(w, 4, 2)
	putChar(w, 30, 5)
	printInt(w, int(newPC.AgeDays/365), 3, 14)
	setCursor(w, 4, 3)
	putChar(w, 31, 5)
	printInt(w, int(newPC.Unk026), 3, 12)
	printMsgAt(200, w, 13, 3, 8)
	w.CurX++
	printInt(w, int(newPC.Level), 3, 9)
	printMsgAt(201, w, 25, 1, 11)
	rank := getMessage(int(newPC.Class)*10 + int(newPC.Rank) + 800)
	setCursor(w, 39-len(rank), 1)
	printString(w, rank, 3)
	printMsgAt(202, w, 25, 2, 14)
	w.CurX++
	printLong(w, newPC.Exp, 10, 6)
	printMsgAt(203, w, 25, 3, 14)
	w.CurX++
	printLong(w, newPC.Kills, 10, 6)
	putGlyphAt(w, 1, 14, 35)
	putGlyphAt(w, 1, 16, 36)
	putGlyphAt(w, 1, 18, 37)
	putGlyphAt(w, 11, 14, 38)
	putGlyphAt(w, 11, 16, 39)
	putGlyphAt(w, 11, 18, 40)
	drawSpellPoints(sheetWin, &newPC)
}

// drawStats (WPCMK 7068) draws the eight stats, the bonus points, HP and stamina.
func drawStats() {
	w := sheetWin
	for i := range 8 {
		printMsgAt(i+204, w, 1, i+5, 5)
		w.CurX++
		printInt(w, int(newPC.Stats[i]), 2, 1)
	}
	if bonusPoints >= 0 {
		setCursor(w, 10, 11)
		printMessage(1107, w, -7)
		w.CurX++
		printInt(w, bonusPoints, 2, 7)
	}
	printMsgAt(212, w, 10, 5, 4)
	w.CurX++
	printInt(w, int(newPC.HP), 4, 6)
	putChar(w, 47, 8)
	setCursor(w, 14, 6)
	printInt(w, int(newPC.HPMax), 4, 6)
	printMsgAt(213, w, 10, 8, 4)
	w.CurX += 2
	printInt(w, int(newPC.Stamina), 3, 6)
	putChar(w, 47, 8)
	setCursor(w, 15, 9)
	printInt(w, int(newPC.StaminaMax), 3, 6)
}

// menuSelect (WPCMK 4800) lays out the enabled messages firstMsg+i in columns of rows
// entries colWidth apart from (x0, y0) and lets the player pick one from entry sel on; the
// chosen message's index. enabled ends with -1.
func menuSelect(win *Window, firstMsg int, enabled []int, x0, y0, colWidth, rows, color, sel int) int {
	var labels []string
	var items []int
	for item := 0; enabled[item] != -1; item++ {
		if enabled[item] != 1 {
			continue
		}
		n := len(labels)
		x, y := n/rows*colWidth+x0, n%rows+y0
		setCursor(win, x, y)
		t := getMessage(firstMsg + item)
		if !mousePresent && sel == n {
			printString(win, t, -color)
		} else {
			printStringHi(win, t, 3)
		}
		labels, items = append(labels, t), append(items, item)
		setClickRegion(n, win.X+byte(x), win.Y+byte(y), byte(len(t)+int(win.X)+x-1), win.Y+byte(y))
	}
	n := len(labels)
	setClickRegion(n, 255, 0, 0, 0)
	at := func() { setCursor(win, sel/rows*colWidth+x0, sel%rows+y0) }
	for {
		if !mousePresent {
			at()
			printString(win, labels[sel], -color)
		}
		item := getMenuInput(menuKeys, clickRegions[:])
		if menuClicked == 1 {
			sel = item
			item = 5
		}
		plain := func() { at(); printStringHi(win, labels[sel], 3) }
		switch item {
		case 1:
			if sel-rows >= 0 {
				plain()
				sel -= rows
			}
		case 2:
			if sel%rows > 0 {
				plain()
				sel--
			}
		case 3:
			if sel+rows < n {
				plain()
				sel += rows
			}
		case 4:
			if sel%rows < rows-1 && sel+1 < n {
				plain()
				sel++
			}
		}
		if item == 5 {
			return items[sel]
		}
	}
}

// drawListBox (WPCMK 619b, WPCVW 719f) draws the scroll bar of a 7-row list window.
func drawListBox(win *Window, show bool) {
	fillWindow(win, 32, 3)
	deferWindowDraw(win)
	putGlyphAt(win, 0, 0, 69)
	for i := 1; i < 6; i++ {
		putGlyphAt(win, 0, i, 71)
	}
	putGlyphAt(win, 0, 6, 70)
	resumeWindowDraw(win)
	if show {
		refreshWindow(win)
	}
}

// findFreeRosterSlot (WPCMK 9558) is the first free roster slot, or -1.
func findFreeRosterSlot() int {
	for i := 0; i < int(pcfileHdr.Count); i++ {
		if rosterSlotUsed[i] == 0 {
			return i
		}
	}
	return -1
}

// isNameTaken (WPCMK 9575) is -1 if a roster character has name, else 0.
func isNameTaken(name string) int {
	for i := 0; i < int(pcfileHdr.Count); i++ {
		if rosterSlotUsed[i] > 0 {
			var pc PC
			pcfileRecordIO(&pc, i, false)
			if cstr(pc.Name[:]) == name {
				return -1
			}
		}
	}
	return 0
}

// showErrorMsg (WPCMK 95bf) prints message msg in the prompt at (x, y), beeps and waits.
func showErrorMsg(msg, x, y int) {
	setCursor(promptWin, x, y)
	printMessage(msg, promptWin, 18)
	playSound(0)
	waitSecondsOrEnter(messageDelay)
}

// drawCreationScreen (WPCMK 95f7) clears the screens and draws the sheet's frame.
func drawCreationScreen(sheet, prompt *Window) {
	fillWindow(sheet, 0, 1)
	fillWindow(prompt, 32, 3)
	fillWindow(menuWin, 32, 3)
	deferWindowDraw(sheet)
	drawCreationFrame(sheet)
	resumeWindowDraw(sheet)
	refreshAllWindows()
}

// readName prints message msg in the prompt and reads a name of up to 7 characters.
func readName(msg int) string {
	setCursor(promptWin, 1, 1)
	printMessage(msg, promptWin, 18)
	x, y := int(promptWin.CurX), int(promptWin.CurY)
	clearWindowArea(promptWin, x, y, x+7, y)
	setCursor(promptWin, x, y)
	return inputText(promptWin, 7, 5, 0)
}

// doCreateCharacter (WPCMK 9656) asks for a name and makes a character in a free slot.
func doCreateCharacter() {
	drawCreationScreen(sheetWin, promptWin)
	slot := findFreeRosterSlot()
	if slot == -1 {
		return
	}
	name := readName(1100)
	maxBonus = false
	if name == "" {
		return
	}
	if isNameTaken(name) == -1 {
		showErrorMsg(1102, 6, 2)
		return
	}
	createCharacter(name, slot)
}

// loadAndShowCharacter (WPCMK 972a) shows roster character slot on the sheet.
func loadAndShowCharacter(slot int) {
	drawCreationScreen(sheetWin, promptWin)
	pcfileRecordIO(&newPC, slot, false)
	bonusPoints = -1
	loadPortrait(int(newPC.Portrait), 0)
	drawPortrait(sheetWin, 0, 1, 1)
	drawSexRaceClass()
	drawCharacterSheet()
	drawStats()
}

// wpcmkWaitForKey (WPCMK 9778 wait_for_key) waits for Enter or a click.
func wpcmkWaitForKey() {
	fillWindow(promptWin, 32, 3)
	refreshWindow(promptWin)
	setCursor(promptWin, 12, 1)
	printMessage(1110, promptWin, 18)
	for {
		if checkScreenClick() {
			return
		}
		if keyAvailable(1) && byte(getKey(1)) == 13 {
			return
		}
	}
}

// yesNo asks the yes/no menu after message msg in the prompt, from entry sel; true for yes.
func yesNo(firstMsg, sel int) bool {
	return menuSelect(promptWin, firstMsg, []int{1, 1, -1}, int(promptWin.CurX)+1, 1, 4, 1, 5, sel) == 0
}

// deleteCharacter (WPCMK 97e1) deletes roster character slot if the player says so.
func deleteCharacter(slot int) {
	loadAndShowCharacter(slot)
	setCursor(promptWin, 5, 1)
	printMessage(1125, promptWin, 18)
	if yesNo(1126, 1) {
		rosterSlotUsed[slot] = 0
		writePcfileHeader()
	}
}

// renameCharacter (WPCMK 989b) gives roster character slot a new name.
func renameCharacter(slot int) {
	loadAndShowCharacter(slot)
	name := readName(1128)
	if name == "" {
		return
	}
	if isNameTaken(name) == -1 {
		showErrorMsg(1102, 6, 2)
		return
	}
	newPC.Name = [8]uint8{}
	copy(newPC.Name[:], name)
	pcfileRecordIO(&newPC, slot, true)
	clearWindowArea(sheetWin, 4, 1, 11, 1)
	drawSexRaceClass()
	wpcmkWaitForKey()
}

// changePortrait (WPCMK 9986) picks a new portrait for roster character slot.
func changePortrait(slot int) {
	loadAndShowCharacter(slot)
	old := newPC.Portrait
	choosePortrait()
	if newPC.Portrait != old {
		pcfileRecordIO(&newPC, slot, true)
		wpcmkWaitForKey()
	}
}

// viewCharacterWpcmk (WPCMK 99c0 view_character) shows roster character slot.
func viewCharacterWpcmk(slot int) {
	loadAndShowCharacter(slot)
	wpcmkWaitForKey()
}

// drawRosterList (WPCMK 99d1) draws seven rows of the roster list around top, the middle
// one highlighted when highlight is 1.
func drawRosterList(win *Window, top, highlight int) {
	drawListBox(win, false)
	deferWindowDraw(win)
	for row := -3; row <= 3; row++ {
		if top+row < 0 || top+row >= len(rosterList) {
			continue
		}
		e := &rosterList[top+row]
		setCursor(win, 1, row+3)
		if highlight == 1 && row == 0 {
			printString(win, e.name, 5)
			for win.CurX < 9 {
				putChar(win, 32, 1)
			}
			putChar(win, getMessage(e.sex + 140)[0], 7)
			putChar(win, 45, 9)
			printString(win, getMessage(e.race+100), 6)
			putChar(win, 32, 1)
			printString(win, getMessage(e.class+120), 3)
			continue
		}
		printStringHi(win, e.name, 3)
		for win.CurX < 9 {
			putCharHi(win, 32, 3)
		}
		putCharHi(win, getMessage(e.sex + 140)[0], 3)
		putCharHi(win, 45, 3)
		printStringHi(win, getMessage(e.race+100), 3)
		win.CurX++
		printStringHi(win, getMessage(e.class+120), 3)
	}
	resumeWindowDraw(win)
	refreshWindow(win)
}

// selectRosterCharacter (WPCMK 9c04) lets the player pick a roster character under title
// msg; its place in rosterList, or -1.
func selectRosterCharacter(win, listWin *Window, titleMsg int) int {
	rosterList = rosterList[:0]
	for i := 0; i < int(pcfileHdr.Count); i++ {
		if rosterSlotUsed[i] == 1 {
			pcfileRecordIO(&newPC, i, false)
			rosterList = append(rosterList, pclistEntry{cstr(newPC.Name[:]), int(newPC.Sex), int(newPC.Race), int(newPC.Class), i})
		}
	}
	n := len(rosterList)
	if n < 1 {
		return -1 // the C returns no value; the menu only offers this with a character
	}
	fillWindow(win, 32, 3)
	fillWindow(listWin, 32, 3)
	deferWindowDraw(win)
	title := getMessage(titleMsg)
	setCursor(win, 20-(len(title)+1)/2, 1)
	printStringHi(win, title, 3)
	resumeWindowDraw(win)
	drawListBox(listWin, false)
	refreshAllWindows()
	cancel := getMessage(1124)
	sel, onCancel := 0, false
	lx, ly := listWin.X, listWin.Y
	setClickRegion(0, lx+1, ly+3, lx+19, ly+3)
	setClickRegion(1, lx, ly, lx, ly)
	setClickRegion(2, lx, ly+6, lx, ly+6)
	setClickRegion(3, win.X+17, win.Y+3, byte(len(cancel)+int(win.X)+16), win.Y+3)
	setClickRegion(4, 255, 0, 0, 0)
	for {
		setCursor(win, 17, 3)
		if onCancel {
			printString(win, cancel, -5)
			drawRosterList(listWin, sel, 0)
		} else {
			printStringHi(win, cancel, 3)
			drawRosterList(listWin, sel, 1)
		}
		k := getMenuInput(menuKeys, clickRegions[:])
		if menuClicked == 1 {
			switch k {
			case 0:
				k = 5
			case 1:
				k = 2
			case 2:
				k = 4
			case 3:
				onCancel = true
				k = 5
			}
		}
		switch k {
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
			} else if sel+1 < n {
				sel++
			}
		}
		if k == 5 {
			break
		}
	}
	if onCancel {
		return -1
	}
	return sel
}

// characterMenu (WPCMK 9f44) is the character menu: create, review, delete, rename,
// change portrait, leave.
func characterMenu() {
	menuKeys = getMessage(1000)
	sheetWin = createWindow(0, 0, 40, 20, 20, -1, 0, false)
	promptWin = createWindow(0, 20, 40, 5, 19, -1, 0, false)
	menuWin = createWindow(21, 7, 19, 13, 21, -1, 0, false)
	for {
		drawCreationScreen(sheetWin, promptWin)
		enabled := []int{1, 1, 1, 1, 1, 1, -1}
		if findFreeRosterSlot() == -1 {
			enabled[0] = 0
		}
		any := 0
		for i := 0; i < int(pcfileHdr.Count); i++ {
			if rosterSlotUsed[i] == 1 {
				any = 1
			}
		}
		enabled[1], enabled[2], enabled[3], enabled[4] = any, any, any, any
		choice := menuSelect(promptWin, 1130, enabled, 4, 1, 12, 2, 5, countEnabledItems(enabled, 5))
		pick := func(msg int, f func(int)) {
			if sel := selectRosterCharacter(promptWin, menuWin, msg); sel != -1 {
				f(rosterList[sel].slot)
			}
		}
		switch choice {
		case 0:
			doCreateCharacter()
		case 1:
			pick(1129, viewCharacterWpcmk)
		case 2:
			pick(1121, deleteCharacter)
		case 3:
			pick(1122, renameCharacter)
		case 4:
			pick(1123, changePortrait)
		}
		if choice == 5 {
			break
		}
	}
	destroyWindow(menuWin, false)
	destroyWindow(promptWin, false)
	destroyWindow(sheetWin, false)
}
