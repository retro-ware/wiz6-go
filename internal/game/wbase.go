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

// wbaseMain (WBASE ovmain): 7 resets the party screen, 4 runs the master options menu,
// 24 the game configuration.
func wbaseMain() {
	if gameState == 7 {
		resetPartyScreen()
	}
	if gameState == 4 {
		doMasterOptions()
	}
	if gameState == 24 {
		doGameConfiguration()
		gameState = int(returnCmd)
	}
}

// charIndex (ROOT char_index, asm) is the position of c in keys, or -1.
func charIndex(keys string, c int) int { return strings.IndexByte(keys, byte(c)) }

// campAnimStep is one step of the camp scene's idle loop: every tenth step the next frame,
// otherwise two delay ticks.
func campAnimStep() {
	if campAnimTick == 0 {
		animPhase = (animPhase + 1) % 2
		drawCampScene(int(animPhase), campAnimDx, campAnimDy)
		drvShowWindow(pictureWin)
	} else {
		delayTick()
		delayTick()
	}
	campAnimTick = (campAnimTick + 1) % 10
}

// waitAnimatedCamp (WBASE 4617) waits ticks*5 steps with the camp scene running; a click or
// Enter ends it early.
func waitAnimatedCamp(ticks int) {
	flushKeys()
	for ticks *= 5; ticks > 0; ticks-- {
		if checkScreenClick() {
			return
		}
		if keyAvailable(1) && byte(getKey(1)) == 13 {
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
		if keyAvailable(1) {
			if c := charIndex(keys, getKey(1)&0xff); c >= 0 {
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

// getMenuInput (WBASE 73c0 get_menu_choice, WPCMK 45fc get_menu_input) is
// getMenuChoiceAnimated without the animation.
func getMenuInput(keys string, regions []byte) int {
	menuClicked = 0
	for {
		if keyAvailable(1) {
			if c := charIndex(keys, getKey(1)&0xff); c >= 0 {
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

// runMenuWbase (WBASE 47c0 run_menu) lays out the enabled messages msgbase+i in columns
// of rows entries colWidth apart from (x0, y0) and lets the player pick one, starting at
// entry sel; the chosen message's index. enabled ends with -1.
func runMenuWbase(win *Window, msgbase int, enabled []int, x0, y0, colWidth, rows, color, sel int) int {
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
	setClickRegion(count, 255, 0, 0, 0)
	resumeWindowDraw(win)
	refreshAllWindows()
	drvShowWindow(g4fac)
	drvShowWindow(pictureWin)
	for {
		if !mousePresent {
			setCursor(win, sel/rows*colWidth+x0, sel%rows+y0)
			printString(win, texts[sel], -color)
		}
		item := getMenuChoiceAnimated(menuKeys, clickRegions[:])
		if menuClicked == 1 {
			sel = item
			item = 5
		}
		redraw := func() { redrawMenuItem(win, sel, colWidth, rows, x0, y0, texts[sel]) }
		switch item {
		case 1:
			if sel-rows >= 0 {
				redraw()
				sel -= rows
			}
		case 2:
			if sel%rows > 0 {
				redraw()
				sel--
			}
		case 3:
			if sel+rows < count {
				redraw()
				sel += rows
			}
		case 4:
			if sel%rows < rows-1 && sel+1 < count {
				redraw()
				sel++
			}
		}
		if item == 5 {
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
	drv.DrawFrames(4, int(p.Pairs[n][0])+dx, int(p.Unk57[n]), lo, hi, 0, false, []byte{byte(n)})
}

// drawCampBackdropChar (WBASE 4cd9) draws frame n of the picture in slot 2 at its place.
func drawCampBackdropChar(n int) {
	p := &monsterPics[2]
	drv.DrawFrames(2, int(p.Pairs[n][0]), int(p.Unk57[n]), 72, 248, 0, false, []byte{byte(n)})
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
	win := createWindow(0, 19, 40, 5, 30, -1, 0, false)
	deferWindowDraw(win)
	fillWindow(win, 32, 3)
	printCentered(win, line1, 1, 18)
	printCentered(win, line2, 2, 18)
	resumeWindowDraw(win)
	refreshAllWindows()
	destroyWindow(win, false)
}

// loadPortrait (WBASE 5072, WPCMK 8ffe, WPCVW a78d) loads portrait number portrait
// (WPORTn.EGA, 14 to a file) as glyphs 72+9*slot.. of font 2.
func loadPortrait(portrait, slot int) {
	if videoFlags&1 == 0 { // only the EGA files are ported
		return
	}
	stopSound()
	b := openData(fmt.Sprintf("WPORT%d.EGA", portrait/14+1), 14)
	g := make([][8][8]uint8, 9)
	if o := portrait % 14 * 32 * 9; o+32*9 <= len(b) {
		for t := range g {
			for y := range 8 {
				for x := range 8 {
					for pl := range 4 {
						if b[o+32*t+8*pl+y]&(0x80>>x) != 0 {
							g[t][y][x] |= 1 << pl
						}
					}
				}
			}
		}
	}
	drv.SetGlyphs(2, slot*9+72, g)
}

// findFreePortraitSlot (WBASE 5190) is the lowest portrait slot the first count members
// do not use.
func findFreePortraitSlot(count int) int {
	for slot := 0; slot < 6; slot++ {
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
	playSound(0)
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
	return getKey(class) & 0xff
}

// inputText (WBASE 7418, WPCMK 4654) reads a line of up to maxlen characters at win's
// cursor: mode 0 takes printable characters, mode 1 digits.
func inputText(win *Window, maxlen, color, mode int) string {
	return readLine(win, maxlen, color, mode, func() int { return getKey(1) & 0xff }, nil)
}

// inputTextAnimated (WBASE 529e) is inputText with the camp scene running; from the
// third character on it sets g4f84.
func inputTextAnimated(win *Window, maxlen, color, mode int) string {
	return readLine(win, maxlen, color, mode, func() int { return waitKeyAnimated(1) }, func(n int) {
		if n > 2 {
			g4f84 = 17210
		}
	})
}

func readLine(win *Window, maxlen, color, mode int, next func() int, typed func(int)) string {
	x, y := int(win.CurX), int(win.CurY)
	var buf []byte
	putChar(win, 97, 1)
	for {
		key := next()
		if key == 13 {
			break
		}
		setCursor(win, x, y)
		putChar(win, 32, 1)
		setCursor(win, x, y)
		switch {
		case key == 8:
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				x--
				setCursor(win, x, y)
				putChar(win, 32, 1)
				setCursor(win, x, y)
			} else {
				playSound(0)
			}
		case len(buf) >= maxlen:
			playSound(0)
		case mode == 0 && key >= 32 && key <= 127, mode == 1 && key >= '0' && key <= '9':
			buf = append(buf, byte(key))
			putChar(win, byte(key), color)
			x++
			if typed != nil {
				typed(len(buf))
			}
		}
		putChar(win, 97, 1)
	}
	setCursor(win, x, y)
	putChar(win, 32, 1)
	return string(buf)
}

// drawGlyphAt (WBASE 5416) draws glyph ch of font 1 at (x, y).
func drawGlyphAt(win *Window, x, y int, ch byte) {
	setCursor(win, x, y)
	putCharHi(win, ch, 1)
}

// checkMagicWord (WBASE 543f) is the copy protection: three random icons and their magic
// word (a spell name, message 4000+); -1 if the word is wrong. The shipped (GOG) build
// copies the typed word over the expected one before comparing, so any word passes; the
// prompt, the icons and the random numbers stay as they were.
func checkMagicWord() int {
	if copyProtectionOK == 1 {
		return 0
	}
	eraseWindow(mainWindow, false)
	win := createWindow(0, 19, 40, 5, 10, -1, 0, false)
	fillWindow(win, 32, 3)
	deferWindowDraw(win)
	printCentered(win, getMessage(1050), 1, 18)
	drawGlyphAt(win, 7, 2, 17)
	drawGlyphAt(win, 33, 2, 19)
	drawGlyphAt(win, 7, 4, 6)
	drawGlyphAt(win, 33, 4, 8)
	for x := 8; x < 33; x++ {
		drawGlyphAt(win, x, 2, 18)
		drawGlyphAt(win, x, 4, 7)
	}
	drawGlyphAt(win, 7, 3, 4)
	drawGlyphAt(win, 33, 3, 5)
	deferWindowDraw(win)
	clearWindowArea(win, 8, 3, 32, 3)
	icon1 := random(8)
	icon3 := random(8)
	icon2 := random(4) + icon3/4<<2
	setCursor(win, 10, 3)
	putChar(win, byte(icon1+104), 4)
	win.CurX++
	putChar(win, byte(icon2+112), 5)
	win.CurX++
	putChar(win, byte(icon3+120), 2)
	inputX := int(win.CurX) + 2
	resumeWindowDraw(win)
	refreshAllWindows()
	clearWindowArea(win, 16, 3, 32, 3)
	setCursor(win, inputX, 3)
	_ = getMessage((icon2<<3+icon3<<6+icon1)%79 + 4000) // the magic word
	word := inputTextAnimated(win, 15, 5, 0)
	g4f84 = 17210
	expected := word
	destroyWindow(win, false)
	if expected != word {
		return -1
	}
	copyProtectionOK = 1
	return 0
}

// setupGameWindows (WBASE 5d39) clears the party picture strip and makes the game's command
// window.
func setupGameWindows() {
	g4f88 = 0
	deferWindowDraw(partyPictureWin)
	for i := range 5 {
		setCursor(partyPictureWin, 19, i)
		putCharHi(partyPictureWin, 27, 3)
	}
	resumeWindowDraw(partyPictureWin)
	eraseWindow(mainWindow, false)
	eraseCursorMark(bottomLineWin, false)
	g4fb0 = createWindow(20, 19, 20, 5, 8, -1, 0, false)
	fillWindow(g4fb0, 32, 3)
	deferWindowDraw(g4fb0)
	// g4f88 is 0 here, so the C's other set of glyphs (for g4f88 != 0) is never drawn.
	drawGlyph2x2(g4fb0, 13, 1, 5, 6, 7, 8)
	drawGlyph2x2(g4fb0, 10, 3, 1, 13, 3, 14)
	drawGlyph4x2(g4fb0, 12, 3, 17)
	drawGlyph2x2(g4fb0, 16, 3, 15, 10, 16, 12)
	for i := range 4 {
		setCursor(g4fb0, 1, i+1)
		printMessage(i+1300, g4fb0, 18)
	}
	resumeWindowDraw(g4fb0)
	curEvent = [4]int16{}
	g4f86 = 0
	mazeA, mazeB = nil, nil
	for i := range g4330 {
		g4330[i], g4326[i] = -1, 0
	}
}

// startGame (WBASE 5f71) checks the copy protection, loads the game and goes to the maze.
func startGame(resume int) {
	if checkMagicWord() == -1 {
		closeGameWindows()
		gameState = 3
		return
	}
	if loadGame(resume) == -1 {
		return
	}
	ensureDiskFor(0, gameStateArg)
	setupGameWindows()
	gameState = 6
}

// drawGauge (WBASE 5fb0, WPCVW 48e8) draws a 3-glyph vertical gauge of cur/max at (x, y),
// glyphs of font 1 from glyph on.
func drawGauge(win *Window, x, y, cur, max, glyph int) {
	level := 0 // max 0 would fault in the C
	if max != 0 {
		level = int(int32(cur)*100/int32(max)/10) + 1
	}
	if level > 10 {
		level = 10
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
	win := g4fba
	if i%2 != 0 {
		win = g4fb8
	}
	p := &party[i]
	x, y := 0, i/2<<2
	setCursor(win, x, y)
	name := cstr(p.Name[:])
	printStringHi(win, name, 3)
	for n := len(name); n < 7; n++ {
		putCharHi(win, 32, 3)
	}
	glyph := int(memberPortrait[i])*9 + 72
	for r := range 3 {
		setCursor(win, x, y+r+1)
		for c := range 3 {
			putCharHi(win, byte(r*3+glyph+c), 2)
		}
	}
	state := conditionGlyph[p.Condition]
	if state == 0 && p.Status[2] > 0 {
		state = 1
	}
	if state < 2 && p.Status[3] > 0 {
		state = 2
	}
	setCursor(win, x+3, y+3)
	putCharHi(win, byte(state+37), 1)
	state = -1
	for s := range 10 {
		if p.Status[s] > 0 && statusGlyph[s] > state {
			state = statusGlyph[s]
		}
	}
	if state != -1 {
		putCharHi(win, byte(state+37), 1)
	} else {
		putCharHi(win, 32, 3)
	}
	setCursor(win, x+3, y+2)
	for k := range byte(2) {
		putCharHi(win, p.Class<<1+k+58, 1)
	}
	setCursor(win, x+3, y+1)
	for h := range 2 {
		if e := p.Equipped[h]; e != 255 && p.Items[e].ID > 0 {
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

// drawFrameGlyph (WBASE 63ce) draws glyph ch of font 2 at (x, y).
func drawFrameGlyph(win *Window, x, y int, ch byte) {
	setCursor(win, x, y)
	putCharHi(win, ch, 2)
}

// drawListFrame (WBASE 63f7) draws the scroll bar of a list window.
func drawListFrame(win *Window, show bool) {
	fillWindow(win, 32, 3)
	deferWindowDraw(win)
	drawFrameGlyph(win, 1, 0, 69)
	for y := 1; y < 4; y++ {
		drawFrameGlyph(win, 1, y, 71)
	}
	drawFrameGlyph(win, 1, 4, 70)
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
	menuKeys = getMessage(1000)
	loadMonsterPicSlot(8, 4)
	campAnimTick = 0
	animPhase = 0
	deferWindowDraw(g4fba)
	deferWindowDraw(g4fb8)
	for curMember = 0; int(curMember) < partyCount; curMember++ {
		drawMemberPanel(int(curMember))
	}
	resumeWindowDraw(g4fba)
	resumeWindowDraw(g4fb8)
	initCampScene()
	sel := 0
	if partyCount >= 6 {
		sel = 3
	}
	for gameState = -1; gameState < 0; {
		showMessageLine(1001)
		eraseWindow(bottomLineWin, false)
		win := createWindow(0, 19, 40, 5, 15, -1, 0, false)
		fillWindow(win, 32, 3)
		enabled := []int{1, 1, 1, 1, 1, 1, 1, 1, 1, -1}
		if partyCount < 1 {
			enabled[1], enabled[2] = 0, 0
		}
		if partyCount < 2 {
			enabled[3] = 0
		}
		enabled[0] = 0
		for i := 0; i < int(pcfileHdr.Count); i++ {
			if rosterSlotUsed[i] == 1 {
				enabled[0] = 1
			}
		}
		if partyCount > 5 {
			enabled[0] = 0
		}
		if partyCount > 0 {
			enabled[4] = 0
		}
		sel = countEnabledItems(enabled, sel)
		choice := runMenuWbase(win, 1002, enabled, 2, 1, 19, 4, 5, sel)
		destroyWindow(win, false)
		sel = 0
		if partyCount >= 6 {
			sel = 3
		}
		switch choice {
		case 0:
			addPartyMember()
			if partyCount == 6 {
				sel = 3
			}
		case 1:
			if curMember = int16(choosePartyMemberWbase(1202)); curMember != -1 {
				gameState = 17
				returnCmd = 4
			}
			sel = 1
		case 2:
			if curMember = int16(choosePartyMemberWbase(1203)); curMember != -1 {
				removePartyMember(int(curMember))
			}
			if partyCount > 0 {
				sel = 2
			}
		case 5:
			removeAllMembers()
			gameState = 16
		case 3:
			for i := 0; i < partyCount; i++ {
				rosterSlotUsed[memberPcfileIndex[i]] = 1
			}
			startGame(0)
		case 4:
			startGame(1)
		case 6:
			returnCmd = 4
			doGameConfiguration()
		case 7:
			removeAllMembers()
			closeGameWindows()
			gameState = 1
		case 8:
			removeAllMembers()
			closeGameWindows()
			gameState = 3
		}
	}
}

// resetPartyScreen (WBASE 7345) frees the maze, clears the party panels and goes back to
// the camp menu.
func resetPartyScreen() {
	mazeA, mazeB = nil, nil
	destroyWindow(g4fb0, false)
	fillWindow(g4fba, 32, 3)
	fillWindow(g4fb8, 32, 3)
	fillWindow(partyPictureWin, 26, 3)
	partyCount = 0
	gameState = 4
}

// closeGameWindows (WBASE 6fe7) takes down the game's windows.
func closeGameWindows() {
	for _, w := range []*Window{g4fac, pictureWin, fullscreenWin, partyPictureWin, g4fba, g4fb8, mainWindow, bottomLineWin} {
		destroyWindow(w, false)
	}
	stopSound()
}
