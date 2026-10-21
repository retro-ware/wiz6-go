package game

import (
	"fmt"
	"strings"

	"github.com/retro-ware/wiz6-go/internal/drv"
)

// WBASE: the camp (master options) menu, the party roster, game configuration and the
// start of a new or saved game.

// WBASE bss.
var (
	campAnimTick int // camp_anim_tick: counts to 10 between camp scene frames
	campAnimDx   int // camp_anim_dx: shift of camp sprite 3
	campAnimDy   int // camp_anim_dy: shift of camp sprite 4

	// menuKeys is message 1000, the menu keys: Escape, left, up, right, down, Enter. WBASE
	// keeps it at ds:5066, WPCMK at ds:541e.
	menuKeys string
)

// The places of the keys in menuKeys, as the menu loops return them.
const (
	menuKeyEscape = iota
	menuKeyLeft
	menuKeyUp
	menuKeyRight
	menuKeyDown
	menuKeyEnter
)

// The camp menu's entries (messages 1002 on).
const (
	campAdd       = iota // add a roster character
	campReview           // view a member
	campRemove           // dismiss a member
	campStart            // start a new game
	campResume           // resume the saved game
	campCreate           // the character menu
	campConfigure        // the game configuration
	campTitle            // the title page
	campQuit             // leave the game
)

// rosterSlotUsed values.
const (
	rosterSlotFree      = 0
	rosterSlotAvailable = 1 // a character that can join
	rosterSlotInParty   = 2 // a character in the party
)

// inputText modes.
const (
	inputPrintable = 0 // ' '..maxInputChar
	inputDigits    = 1 // '0'..'9'
)

const (
	campAnimFrameSteps  = 10    // campAnimStep calls per camp scene frame
	campStepsPerTick    = 5     // campAnimStep calls per waitAnimatedCamp tick
	magicWordTypedValue = 17210 // what inputTextAnimated sets magicWordTyped to (the original's value)
	textCursorGlyph     = 97    // font 1 glyph of readLine's cursor
	maxInputChar        = 127   // the last character inputPrintable takes
	noEquippedItem      = 255   // a PC.Equipped entry for an empty hand
	panelNameWidth      = 7     // the member panel's name field, padded with spaces
	stateGlyphBase      = 37    // font 1 glyph of conditionGlyph/statusGlyph 0
	gaugeLevels         = 10    // drawGauge's full level
	pclistSexColumn     = 10    // drawPcfileList's column after the name
)

// The portraits: WPORTn.EGA holds portraitsPerFile portraits of portraitGlyphs 8x8 glyphs,
// each glyph four EGA planes of 8 rows, a row's leftmost pixel in the top bit. Slot n's
// glyphs are font 2's portraitFirstGlyph+portraitGlyphs*n...
const (
	portraitsPerFile   = 14
	portraitGlyphs     = 9
	portraitFirstGlyph = 72
	egaPlanes          = 4
	glyphBytes         = egaPlanes * 8
	portraitBytes      = portraitGlyphs * glyphBytes
	leftmostPixelBit   = 0x80
)

// wbaseMain (WBASE ovmain): stateResetPartyScreen resets the party screen,
// stateMasterOptions runs the master options menu, stateGameConfiguration the game
// configuration.
func wbaseMain() {
	if gameState == stateResetPartyScreen {
		resetPartyScreen()
	}
	if gameState == stateMasterOptions {
		doMasterOptions()
	}
	if gameState == stateGameConfiguration {
		doGameConfiguration()
		gameState = gameStateID(returnCmd)
	}
}

// charIndex (ROOT char_index, asm) is the position of c in keys, or -1.
func charIndex(keys string, c int) int { return strings.IndexByte(keys, byte(c)) }

// campAnimStep is one step of the camp scene's idle loop: every tenth step the next frame,
// otherwise two delay ticks.
func campAnimStep() {
	if campAnimTick == 0 {
		animPace(mazeAnimPeriod)
		animPhase = (animPhase + 1) % 2
		drawCampScene(int(animPhase), campAnimDx, campAnimDy)
		drvShowWindow(pictureWin)
	} else {
		delayTick()
		delayTick()
	}
	campAnimTick = (campAnimTick + 1) % campAnimFrameSteps
}

// waitAnimatedCamp (WBASE 4617) waits ticks*5 steps with the camp scene running; a click or
// Enter ends it early.
func waitAnimatedCamp(ticks int) {
	flushKeys()
	for ticks *= campStepsPerTick; ticks > 0; ticks-- {
		if checkScreenClick() {
			return
		}
		if keyAvailable(keyClassMenu) && byte(getKey(keyClassMenu)) == keyEnter {
			return
		}
		campAnimStep()
	}
}

// getMenuChoiceAnimated (WBASE 469f) waits for a key in keys or a click in regions with the
// camp scene running. A click sets menuClicked and returns the region index; a key returns
// its place in keys.
func getMenuChoiceAnimated(keys string, regions []byte) int {
	menuClicked = 0
	for {
		if keyAvailable(keyClassMenu) {
			if c := charIndex(keys, int(byte(getKey(keyClassMenu)))); c >= 0 {
				return c
			}
			continue
		}
		choice := checkClickRegions(regions)
		if clickFlag {
			menuClicked = 1
			return choice - 1
		}
		campAnimStep()
	}
}

// menuInput is getMenuInput on menuKeys and the click regions.
func menuInput() int { return getMenuInput(menuKeys, clickRegions[:]) }

// getMenuInput (WBASE 73c0 get_menu_choice, WPCMK 45fc get_menu_input) is
// getMenuChoiceAnimated without the animation.
func getMenuInput(keys string, regions []byte) int {
	menuClicked = 0
	for {
		if keyAvailable(keyClassMenu) {
			if c := charIndex(keys, int(byte(getKey(keyClassMenu)))); c >= 0 {
				return c
			}
			continue
		}
		choice := checkClickRegions(regions)
		if clickFlag {
			menuClicked = 1
			return choice - 1
		}
	}
}

// showMessageLine (WBASE 473a) prints message msg centred on the message line.
func showMessageLine(msg int) {
	text := getMessage(msg)
	eraseWindow(mainWindow, false)
	deferWindowDraw(mainWindow)
	printCentered(mainWindow, text, 0, 18)
	resumeWindowDraw(mainWindow)
}

// redrawMenuItem (WBASE 4785) redraws menu entry item in its normal colour.
func redrawMenuItem(win *Window, item, colWidth, rows, x0, y0 int, text string) {
	setCursor(win, item/rows*colWidth+x0, item%rows+y0)
	printStringHi(win, text, 3)
}

// runCampMenu (run_menu, WBASE 47c0) is menuSelect with the camp scene animating, no
// first highlight and a full refresh. It lays out the enabled messages msgbase+i in columns
// of rows entries colWidth apart from (x0, y0) and lets the player pick one, starting at
// entry sel; the chosen message's index. enabled ends with -1.
func runCampMenu(win *Window, msgbase int, enabled []int, x0, y0, colWidth, rows, color, sel int) int {
	var texts []string
	var items []int
	deferWindowDraw(win)
	for item := 0; enabled[item] != -1; item++ {
		if enabled[item] != 1 {
			continue
		}
		n := len(texts)
		x, y := n/rows*colWidth+x0, n%rows+y0
		setCursor(win, x, y)
		t := getMessage(msgbase + item)
		printStringHi(win, t, 3)
		texts, items = append(texts, t), append(items, item)
		setClickRegion(n, win.X+byte(x), win.Y+byte(y), byte(len(t)+int(win.X)+x-1), win.Y+byte(y))
	}
	count := len(texts)
	setClickRegion(count, clickRegionsEnd, 0, 0, 0)
	resumeWindowDraw(win)
	refreshAllWindows()
	drvShowWindow(titleStripWin)
	drvShowWindow(pictureWin)
	for {
		if !mousePresent {
			setCursor(win, sel/rows*colWidth+x0, sel%rows+y0)
			printString(win, texts[sel], -color)
		}
		item := getMenuChoiceAnimated(menuKeys, clickRegions[:])
		if menuClicked == 1 {
			sel = item
			item = menuKeyEnter
		}
		redraw := func() { redrawMenuItem(win, sel, colWidth, rows, x0, y0, texts[sel]) }
		switch item {
		case menuKeyLeft:
			if sel-rows >= 0 {
				redraw()
				sel -= rows
			}
		case menuKeyUp:
			if sel%rows > 0 {
				redraw()
				sel--
			}
		case menuKeyRight:
			if sel+rows < count {
				redraw()
				sel += rows
			}
		case menuKeyDown:
			if sel%rows < rows-1 && sel+1 < count {
				redraw()
				sel++
			}
		}
		if item == menuKeyEnter {
			return items[sel]
		}
	}
}

// drawGlyph2x2 (WBASE 4ac2) draws four glyphs of font 2 as a 2x2 block at (x, y).
func drawGlyph2x2(win *Window, x, y int, a, b, c, d byte) {
	setCursor(win, x, y)
	putCharHi(win, a, 2)
	putCharHi(win, b, 2)
	setCursor(win, x, y+1)
	putCharHi(win, c, 2)
	putCharHi(win, d, 2)
}

// drawGlyph4x2 (WBASE 4b35) draws the eight glyphs of font 2 from first on as a 4x2 block.
func drawGlyph4x2(win *Window, x, y int, first byte) {
	setCursor(win, x, y)
	for i := range byte(4) {
		putCharHi(win, first+i, 2)
	}
	setCursor(win, x, y+1)
	for i := range byte(4) {
		putCharHi(win, first+i+4, 2)
	}
}

// loadMonsterPicSlot (WBASE 4bb2) loads monster mon's picture into slot unless it is there.
func loadMonsterPicSlot(mon, slot int) {
	stopSound()
	if picSlotMonster[slot] == mon {
		return
	}
	loadMonsterPic(mon, slot, 21)
}

// drawCampSprite (WBASE 4c96) draws frame n of the camp picture (slot 4) at its place
// shifted by dx, clipped to columns lo..hi.
func drawCampSprite(n, dx, lo, hi int) {
	p := &monsterPics[4]
	drv.DrawFrames(4, int(p.FrameX[n][0])+dx, int(p.FrameY[n]), lo, hi, 0, false, []byte{byte(n)})
}

// drawCampBackdropChar (WBASE 4cd9) draws frame n of the picture in slot 2 at its place.
func drawCampBackdropChar(n int) {
	p := &monsterPics[2]
	drv.DrawFrames(2, int(p.FrameX[n][0]), int(p.FrameY[n]), 72, 248, 0, false, []byte{byte(n)})
}

// drawCampScene (WBASE 4d1b) draws the camp scene into the view for animation frame,
// with sprites 3 and 4 shifted by dx and dy.
func drawCampScene(frame, dx, dy int) {
	drvFillBack(pictureWin, 0)
	for _, t := range [][3]int{
		{123, 1, 123}, {124, 1, 124}, {125, 1, 125}, {20, 1, 16}, {21, 1, 17}, {22, 1, 18},
		{16, 1, 20}, {17, 1, 21}, {18, 1, 22}, {151, 1, 151}, {152, 1, 152}, {153, 1, 153},
		{25, 0, -1}, {28, 0, -1}, {31, 0, -1}, {34, 0, -1},
	} {
		drvMazeTile(t[0], t[1], t[2])
	}
	drawCampBackdropChar(15)
	drawCampBackdropChar(12)
	drawCampSprite(1, 0, 72, 248)
	drawCampSprite(2, 0, 72, 248)
	drawCampSprite(3, dx, 128, 192)
	drawCampSprite(4, dy, 128, 192)
	if frame != 0 {
		drawCampSprite(5, 0, 72, 248)
		drawCampSprite(6, 0, 72, 248)
	}
}

// initCampScene (WBASE 4ee8) resets the camp animation and draws the scene.
func initCampScene() {
	for _, t := range []int{346, 349, 352, 361, 355, 358} {
		drvMazeTile(t, 0, -1)
	}
	animPhase = 0
	campAnimDx, campAnimDy = 0, 0
	drawCampScene(int(animPhase), campAnimDx, campAnimDy)
}

// campShift is how far the camp doors open per step (ROOT g0022).
const campShift = 8

// animateAndShowTwoLines (WBASE 4f79) opens the camp doors, then shows two lines in a
// popup.
func animateAndShowTwoLines(line1, line2 string) {
	for step := 1; step < 9; step++ {
		animPhase = (animPhase + 1) % 2
		campAnimDx = -campShift * step / 2
		campAnimDy = step * campShift / 2
		drawCampScene(int(animPhase), campAnimDx, campAnimDy)
		drvShowWindow(pictureWin)
		playSound(3)
	}
	waitAnimatedCamp(5)
	eraseWindow(mainWindow, false)
	win := createWindow(0, 19, 40, 5, 30, borderNone, 0, false)
	deferWindowDraw(win)
	fillWindow(win, ' ', 3)
	printCentered(win, line1, 1, 18)
	printCentered(win, line2, 2, 18)
	resumeWindowDraw(win)
	refreshAllWindows()
	destroyWindow(win, false)
}

// loadPortrait (WBASE 5072, WPCMK 8ffe, WPCVW a78d) loads portrait number portrait
// (WPORTn.EGA, 14 to a file) as glyphs 72+9*slot.. of font 2.
func loadPortrait(portrait, slot int) {
	if videoFlags&videoEGA == 0 { // only the EGA files are ported
		return
	}
	stopSound()
	b := openData(fmt.Sprintf("WPORT%d.EGA", portrait/portraitsPerFile+1), 14)
	g := make([][8][8]uint8, portraitGlyphs)
	if o := portrait % portraitsPerFile * portraitBytes; o+portraitBytes <= len(b) {
		for t := range g {
			for y := range 8 {
				for x := range 8 {
					for pl := range egaPlanes {
						if b[o+glyphBytes*t+8*pl+y]&(leftmostPixelBit>>x) != 0 {
							g[t][y][x] |= 1 << pl
						}
					}
				}
			}
		}
	}
	drv.SetGlyphs(2, slot*portraitGlyphs+portraitFirstGlyph, g)
}

// findFreePortraitSlot (WBASE 5190) is the lowest portrait slot the first count members
// do not use.
func findFreePortraitSlot(count int) int {
	for slot := 0; slot < len(party); slot++ {
		used := false
		for i := 0; i < count; i++ {
			if int(memberPortrait[i]) == slot {
				used = true
			}
		}
		if !used {
			return slot
		}
	}
	return 0
}

// showStatusText (WBASE 51e4) prints text on the message line.
func showStatusText(text string) {
	eraseWindow(mainWindow, true)
	printCentered(mainWindow, text, 0, 18)
	flushKeys()
}

// showErrorText (WBASE 5211) prints an error, beeps and waits (with the camp scene running
// if animated).
func showErrorText(text string, animated bool) {
	showStatusText(text)
	playSound(soundBeep)
	if animated {
		waitAnimatedCamp(12)
	} else {
		waitSecondsOrEnter(12)
	}
}

// waitKeyAnimated (WBASE 523f) waits for a key of class with the camp scene running.
func waitKeyAnimated(class int) int {
	for !keyAvailable(class) {
		campAnimStep()
	}
	return int(byte(getKey(class)))
}

// inputText (WBASE 7418, WPCMK 4654, WPCVW 9459) reads a line of up to maxlen characters at win's
// cursor: mode inputPrintable takes printable characters, inputDigits digits.
func inputText(win *Window, maxlen, color, mode int) string {
	return readLine(win, maxlen, color, mode, func() int { return int(byte(getKey(keyClassMenu))) }, nil)
}

// inputTextAnimated (WBASE 529e) is inputText with the camp scene running; from the
// third character on it sets magicWordTyped.
func inputTextAnimated(win *Window, maxlen, color, mode int) string {
	return readLine(win, maxlen, color, mode, func() int { return waitKeyAnimated(keyClassMenu) }, func(n int) {
		if n > 2 {
			magicWordTyped = magicWordTypedValue
		}
	})
}

// readLine is the input loop of inputText: next reads a key, typed (if any) is told the
// length after each character.
func readLine(win *Window, maxlen, color, mode int, next func() int, typed func(int)) string {
	x, y := int(win.CurX), int(win.CurY)
	var buf []byte
	putChar(win, textCursorGlyph, 1)
	for {
		key := next()
		if key == keyEnter {
			break
		}
		setCursor(win, x, y)
		putChar(win, ' ', 1)
		setCursor(win, x, y)
		switch {
		case key == keyLeft: // Backspace
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				x--
				setCursor(win, x, y)
				putChar(win, ' ', 1)
				setCursor(win, x, y)
			} else {
				playSound(soundBeep)
			}
		case len(buf) >= maxlen:
			playSound(soundBeep)
		case mode == inputPrintable && key >= ' ' && key <= maxInputChar, mode == inputDigits && key >= '0' && key <= '9':
			buf = append(buf, byte(key))
			putChar(win, byte(key), color)
			x++
			if typed != nil {
				typed(len(buf))
			}
		}
		putChar(win, textCursorGlyph, 1)
	}
	setCursor(win, x, y)
	putChar(win, ' ', 1)
	return string(buf)
}

// setupGameWindows (WBASE 5d39) clears the party picture strip and makes the game's command
// window.
func setupGameWindows() {
	padSidestep = 0
	deferWindowDraw(partyPictureWin)
	for i := range 5 {
		setCursor(partyPictureWin, 19, i)
		putCharHi(partyPictureWin, 27, 3)
	}
	resumeWindowDraw(partyPictureWin)
	eraseWindow(mainWindow, false)
	eraseCursorMark(bottomLineWin, false)
	movePadWin = createWindow(20, 19, 20, 5, 8, borderNone, 0, false)
	fillWindow(movePadWin, ' ', 3)
	deferWindowDraw(movePadWin)
	// padSidestep is 0 here, so the C's other set of glyphs (for padSidestep != 0) is never drawn.
	drawGlyph2x2(movePadWin, 13, 1, 5, 6, 7, 8)
	drawGlyph2x2(movePadWin, 10, 3, 1, 13, 3, 14)
	drawGlyph4x2(movePadWin, 12, 3, 17)
	drawGlyph2x2(movePadWin, 16, 3, 15, 10, 16, 12)
	for i := range 4 {
		setCursor(movePadWin, 1, i+1)
		printMessage(i+msgPressReturnBase, movePadWin, 18)
	}
	resumeWindowDraw(movePadWin)
	curEvent = [4]int16{}
	secretSpotted = 0
	mazeA, mazeB = nil, nil
	for i := range 3 {
		picSlotMonster[4+i], picSlotLoaded[4+i] = -1, false
	}
}

// startGame (WBASE 5f71) loads the game and goes to the maze. The C first asks for the
// copy protection's magic word (WBASE 543f); the port leaves it out.
func startGame(resume int) {
	if loadGame(resume) == -1 {
		return
	}
	ensureDiskFor(0, gameStateArg)
	setupGameWindows()
	gameState = stateLoadMap
}

// drawGauge (WBASE 5fb0, WPCVW 48e8) draws a 3-glyph vertical gauge of cur/max at (x, y),
// glyphs of font 1 from glyph on.
func drawGauge(win *Window, x, y, cur, max, glyph int) {
	level := 0 // max 0 would fault in the C
	if max != 0 {
		level = int(int16(int32(cur)*100/int32(max)/10) + 1) // (int) cast, then a 16-bit +1
	}
	if level > gaugeLevels {
		level = gaugeLevels
	}
	if cur == 0 {
		level = 0
	}
	g := [3]int{glyph + 9, glyph + 4, glyph}
	switch {
	case level <= 3:
		g[2] = glyph + level
	case level-3 <= 4:
		g[2] = glyph + 3
		g[1] = glyph + level - 3 + 4
	default:
		g[2] = glyph + 3
		g[1] = glyph + 8
		g[0] = glyph + level - 7 + 9
	}
	for i := range 3 {
		setCursor(win, x, y+i)
		putCharHi(win, byte(g[i]), 1)
	}
}

// drawMemberPanel (WBASE 6091) draws party member i in a side panel: name, portrait,
// condition, status, class, hands and the HP and stamina gauges.
func drawMemberPanel(i int) {
	if dtLog != nil { // stubbed in the difftest
		return
	}
	win := leftPanelWin
	if i%2 != 0 {
		win = rightPanelWin
	}
	p := &party[i]
	x, y := 0, i/2<<2
	setCursor(win, x, y)
	name := cstr(p.Name[:])
	printStringHi(win, name, 3)
	for n := len(name); n < panelNameWidth; n++ {
		putCharHi(win, ' ', 3)
	}
	glyph := int(memberPortrait[i])*portraitGlyphs + portraitFirstGlyph
	for r := range 3 {
		setCursor(win, x, y+r+1)
		for c := range 3 {
			putCharHi(win, byte(r*3+glyph+c), 2)
		}
	}
	state := conditionGlyph[p.Condition]
	if state == 0 && p.Status[statusAsleep] > 0 {
		state = 1
	}
	if state < 2 && p.Status[statusParalyzed] > 0 {
		state = 2
	}
	setCursor(win, x+3, y+3)
	putCharHi(win, byte(state+stateGlyphBase), 1)
	state = -1
	for s := range len(statusGlyph) {
		if p.Status[s] > 0 && statusGlyph[s] > state {
			state = statusGlyph[s]
		}
	}
	if state != -1 {
		putCharHi(win, byte(state+stateGlyphBase), 1)
	} else {
		putCharHi(win, ' ', 3)
	}
	setCursor(win, x+3, y+2)
	for k := range byte(2) {
		putCharHi(win, p.Class<<1+k+58, 1)
	}
	setCursor(win, x+3, y+1)
	for h := range 2 {
		if e := p.Equipped[h]; e != noEquippedItem && p.Items[e].ID > 0 {
			putCharHi(win, p.Items[e].Icon+1, 4)
		} else {
			putCharHi(win, byte(h+37), 4)
		}
	}
	drawGauge(win, x+5, y+1, int(p.HP), int(p.HPMax), 86)
	drawGauge(win, x+6, y+1, int(p.Stamina), int(p.StaminaMax), 99)
}

// showTitleText (WBASE 6372) prints text at column x of the message line.
func showTitleText(text string, x int) {
	eraseWindow(mainWindow, false)
	eraseCursorMark(mainWindow, false)
	deferWindowDraw(mainWindow)
	setCursor(mainWindow, x, 0)
	printStringHi(mainWindow, text, 3)
	mainWindow.CurX++
	resumeWindowDraw(mainWindow)
}

// drawListFrame (WBASE 63f7, WTREA 7549, ...) draws the scroll bar of a list window.
func drawListFrame(win *Window, show bool) {
	fillWindow(win, ' ', 3)
	deferWindowDraw(win)
	putGlyphAt(win, 1, 0, 69)
	for y := 1; y < 4; y++ {
		putGlyphAt(win, 1, y, 71)
	}
	putGlyphAt(win, 1, 4, 70)
	resumeWindowDraw(win)
	if show {
		refreshWindow(win)
	}
}

// countEnabledItems (WBASE 705b, WPCMK 47c1 count_enabled_options) counts the enabled
// entries before index n; 0 if the -1 end comes first.
func countEnabledItems(enabled []int, n int) int {
	count := 0
	for i := 0; i != n; i++ {
		if enabled[i] == -1 {
			return 0
		}
		if enabled[i] == 1 {
			count++
		}
	}
	return count
}

// doMasterOptions (WBASE 709a) is the camp menu: add, review and dismiss members, start a
// new or saved game, the character menu, configuration, the title page and quit.
func doMasterOptions() {
	menuKeys = getMessage(msgCampKeys)
	loadMonsterPicSlot(8, 4)
	campAnimTick = 0
	animPhase = 0
	deferWindowDraw(leftPanelWin)
	deferWindowDraw(rightPanelWin)
	for curMember = 0; int(curMember) < partyCount; curMember++ {
		drawMemberPanel(int(curMember))
	}
	resumeWindowDraw(leftPanelWin)
	resumeWindowDraw(rightPanelWin)
	initCampScene()
	sel := 0
	if partyCount >= len(party) {
		sel = campStart
	}
	for gameState = -1; gameState < 0; {
		showMessageLine(msgMasterOptions)
		eraseWindow(bottomLineWin, false)
		win := createWindow(0, 19, 40, 5, 15, borderNone, 0, false)
		fillWindow(win, ' ', 3)
		enabled := []int{1, 1, 1, 1, 1, 1, 1, 1, 1, -1}
		if partyCount < 1 {
			enabled[campReview], enabled[campRemove] = 0, 0
		}
		if partyCount < 2 {
			enabled[campStart] = 0
		}
		enabled[campAdd] = 0
		for i := 0; i < int(pcfileHdr.Count); i++ {
			if rosterSlotUsed[i] == rosterSlotAvailable {
				enabled[campAdd] = 1
			}
		}
		if partyCount >= len(party) {
			enabled[campAdd] = 0
		}
		if partyCount > 0 {
			enabled[campResume] = 0
		}
		sel = countEnabledItems(enabled, sel)
		choice := runCampMenu(win, msgCampMenu, enabled, 2, 1, 19, 4, 5, sel)
		destroyWindow(win, false)
		sel = 0
		if partyCount >= len(party) {
			sel = campStart
		}
		switch choice {
		case campAdd:
			addPartyMember()
			if partyCount == len(party) {
				sel = campStart
			}
		case campReview:
			if curMember = int16(choosePartyMemberWbase(msgCampReviewWho)); curMember != -1 {
				gameState = stateViewCharacter
				returnCmd = int16(stateMasterOptions)
			}
			sel = campReview
		case campRemove:
			if curMember = int16(choosePartyMemberWbase(msgDismissWho)); curMember != -1 {
				removePartyMember(int(curMember))
			}
			if partyCount > 0 {
				sel = campRemove
			}
		case campCreate:
			removeAllMembers()
			gameState = stateCreateCharacter
		case campStart:
			for i := 0; i < partyCount; i++ {
				rosterSlotUsed[memberPcfileIndex[i]] = rosterSlotAvailable
			}
			startGame(0)
		case campResume:
			startGame(1)
		case campConfigure:
			returnCmd = int16(stateMasterOptions)
			doGameConfiguration()
		case campTitle:
			removeAllMembers()
			closeGameWindows()
			gameState = stateTitle
		case campQuit:
			removeAllMembers()
			closeGameWindows()
			gameState = stateQuit
		}
	}
}

// resetPartyScreen (WBASE 7345) frees the maze, clears the party panels and goes back to
// the camp menu.
func resetPartyScreen() {
	mazeA, mazeB = nil, nil
	destroyWindow(movePadWin, false)
	fillWindow(leftPanelWin, ' ', 3)
	fillWindow(rightPanelWin, ' ', 3)
	fillWindow(partyPictureWin, 26, 3)
	partyCount = 0
	gameState = stateMasterOptions
}

// closeGameWindows (WBASE 6fe7) takes down the game's windows.
func closeGameWindows() {
	for _, w := range []*Window{titleStripWin, pictureWin, fullscreenWin, partyPictureWin, leftPanelWin, rightPanelWin, mainWindow, bottomLineWin} {
		destroyWindow(w, false)
	}
	stopSound()
}

// MSG.DBS message ids.
const (
	msgPressReturnBase = 1300 // plus 0..3: PRESS, RETURN, FOR, OPTIONS

	msgCampKeys        = 1000
	msgMasterOptions   = 1001
	msgCampMenu        = 1002 // ADD PARTY MEMBER
	msgAddMember       = 1201
	msgCampReviewWho   = 1202 // review_who}
	msgDismissWho      = 1203
	msgAddWho          = 1206
	msgAddCancel       = 1207 // CANCEL
	msgEntering        = 1212
	msgBaneCosmicForge = 1213 // BANE OF THE COSMIC FORGE
)
