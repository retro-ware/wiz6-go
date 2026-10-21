package game

import (
	"strings"

	"github.com/retro-ware/wiz6-go/internal/drv"
)

// WMAZE: event text, dice, skills and the simple cell events (9948..b53d).

// Root data tables the door animation reads (ROOT data 0x0022..0x0052), per view depth.
var (
	doorStep  = [4]int{8, 5, 3, 0}       // g0022
	doorStart = [4]int{48, 30, 18, 0}    // g002a
	doorClipL = [4]int{136, 146, 153, 0} // g0032 (stride 4)
	doorClipR = [4]int{184, 174, 167, 0} // g0034 (stride 4)
)

// eventData is the current cell event's Data, Unk288 and DirBits words.
func eventData() int  { return int(mazeB.Data[cellEventIndex]) }
func event288() int   { return int(mazeB.Unk288[cellEventIndex]) }
func eventBits() int  { return int(uint16(mazeB.DirBits[cellEventIndex])) }
func eventParam() int { return int(mazeB.Param[cellEventIndex]) }

// replace_char_with_string (ROOT 05c1) replaces the first ch of s with repl.
func replaceCharWithString(s, repl string, ch byte) string {
	if i := strings.IndexByte(s, ch); i >= 0 {
		return s[:i] + repl + s[i+1:]
	}
	return s
}

// save_level_flags (WMAZE 98d7) writes the current map's level flags and geometry back.
func saveLevelFlags() {
	scenarioHdr.LevelFlags[gameStateArg] = levelFlags[gameStateArg]
	mapCacheA[gameStateArg] = *mazeA
	mapCacheB[gameStateArg] = *mazeB
}

// load_sound_bank (WMAZE 9948) is the root's loadSound: sound n into slot 14.
// The C fails with 15 for a missing file and 16 for a bad one; loadSoundSlot has one code.
func loadSoundBank(n int) {
	if dtLog != nil { // the difftest stubs the disk load
		curSound = n
		return
	}
	loadSound(n)
}

// try_skill_gain (WMAZE 9a24) may raise member m's skill by one; the higher the level
// times difficulty, the likelier.
func trySkillGain(m, skill, difficulty int) {
	p := &party[m]
	if p.Skills[skill] < 100 && random(int(p.Skills[skill]))-int(int16(int(p.Level)*difficulty)) <= 0 { // the product wraps; the compare does not (sub; jg)
		p.Skills[skill]++
	}
}

// wait_key_animating (WMAZE 9a66) animates the view until a key of class comes.
func waitKeyAnimating(class int) int {
	for !keyAvailable(class) {
		animateStep()
	}
	return int(uint8(getKey(class)))
}

// input_text_m (WMAZE 9aae) reads up to maxLen characters at win's cursor, in colour;
// digits only for mode 1, printable characters for mode 0.
func inputTextAnimating(win *Window, maxLen, color, mode int) string {
	col, row := int(win.CurX), int(win.CurY)
	var buf []byte
	putChar(win, 'a', 1)
	for {
		c := waitKeyAnimating(keyClassMenu)
		if c == keyEnter {
			break
		}
		setCursor(win, col, row)
		putChar(win, ' ', 1)
		setCursor(win, col, row)
		switch {
		case c == keyLeft:
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				col--
				setCursor(win, col, row)
				putChar(win, ' ', 1)
				setCursor(win, col, row)
			} else {
				playSound(soundBeep)
			}
		case len(buf) >= maxLen:
			playSound(soundBeep)
		case mode == inputPrintable && c >= ' ' && c <= maxInputChar, mode == inputDigits && c >= '0' && c <= '9':
			buf = append(buf, byte(c))
			putChar(win, byte(c), color)
			col++
		}
		putChar(win, 'a', 1)
	}
	setCursor(win, col, row)
	putChar(win, ' ', 1)
	return string(buf)
}

// wait_more_prompt_m (WMAZE 9c19) blinks the more mark in the text window's corner until
// Enter or a click.
func waitMorePrompt() {
	if dtLog != nil { // stubbed in the difftest
		return
	}
	blink := 3
	flushKeys()
	for {
		if keyAvailable(keyClassMenu) {
			if int8(getKey(keyClassMenu)) == keyEnter {
				return
			}
			continue
		}
		if checkScreenClick() {
			return
		}
		if mzAnimCounter == 0 {
			animPace(mazeAnimPeriod)
			animPhase = (animPhase + 1) % 2
			redrawMazeView()
			setCursor(textWin, 39, 4)
			if blink < 2 {
				putChar(textWin, ' ', 1)
			} else {
				putCharHi(textWin, 126, 1)
			}
			blink = (blink + 1) % 4
		} else {
			idleStep()
		}
		mzAnimCounter = (mzAnimCounter + 1) % animFrameSteps
	}
}

// end_text_page_m (WMAZE 9cde): a timed pause (timed 1), or the more prompt and a clean page.
func endTextPage(timed int) {
	if dtLog != nil { // stubbed in the difftest
		return
	}
	if timed == 1 {
		waitAnimating(messageDelay)
		return
	}
	waitMorePrompt()
	clearWindow(textWin, true)
	textLine = 0
}

// print_text_line_m (WMAZE 9d0c) runs the control characters at the start of s, then
// prints the rest centred ('$' at the left, '^' at the last line's column).
func printTextLine(s string) {
	for len(s) > 0 {
		k := strings.IndexByte("!%#&]@~", s[0])
		if k < 0 {
			break
		}
		switch k {
		case 0:
			s = s[1:]
			textLine++
		case 1:
			s = s[1:]
			waitAnimating(messageDelay)
		case 2:
			s = s[min(2, len(s)):]
		case 3:
			s = s[1:]
		case 4:
			s = s[1:]
			clearWindow(textWin, true)
			textLine = 0
		case 5:
			s = s[1:]
			endTextPage(0)
		case 6:
			s = s[1:]
			n := 0
			if len(s) > 0 {
				n = int(s[0])
				s = s[1:]
			}
			if n >= loadedSound {
				if n != curSound {
					loadSoundBank(n)
				}
				playSound(loadedSound)
			} else {
				playSound(n)
			}
		}
	}
	if textLine > 4 {
		endTextPage(0)
	}
	col := 20 - (len(s)+1)/2
	if len(s) > 0 {
		switch s[0] {
		case '$':
			s, col = s[1:], 0
		case '^':
			s, col = s[1:], int(textLastColumn)
		}
	}
	setCursor(textWin, col, int(textLine))
	textLastColumn = int16(col)
	textLine++
	printString(textWin, s, 5)
}

// openEventTextWin hides the message lines under the event text window (rows 19..23).
func openEventTextWin(refresh bool) {
	eventScreenDirty = 1
	deferWindowDraw(mainWindow)
	deferWindowDraw(bottomLineWin)
	textWin = createWindow(0, 19, 40, 5, 20, borderNone, 0, refresh)
}

func closeEventTextWin() {
	destroyWindow(textWin, false)
	resumeWindowDraw(mainWindow)
	resumeWindowDraw(bottomLineWin)
}

// show_message_text_m (WMAZE 9e51) shows messages msg, msg+1, ... up to the first
// missing one in the text window; timed 1 ends with a pause instead of the more prompt.
func showMessageText(msg, timed int) {
	if dtLog != nil { // the difftest stubs it
		return
	}
	textLastColumn, textLine = 0, 0
	openEventTextWin(true)
	for {
		s := getMessage(msg)
		if msgNotFound {
			break
		}
		printTextLine(s)
		msg++
	}
	endTextPage(timed)
	closeEventTextWin()
}

// roll_dice (WMAZE 9ef6) is d.Base plus Count rolls of 1..Sides.
func rollDice(d Dice) int {
	n := int(int16(d.Base))
	for c := d.Count; c != 0; c-- {
		n += random(int(d.Sides)) + 1
	}
	return int(int16(n)) // C int wraps
}

// format_dice (WMAZE 9f25) makes count dice of sides plus base.
func formatDice(count, sides, base int) Dice {
	return Dice{Base: int16(base), Count: uint8(count), Sides: uint8(sides)}
}

// randomOpenDir (WMAZE 9f54) is a random direction whose open[] is set, trying on from a
// random start.
func randomOpenDir(open *[4]int) int {
	d := random(4)
	for open[d] == 0 {
		d = (d + 1) % dirCount
	}
	return d
}

// spinnerEvent (WMAZE 9f8c): the spinner/pusher event. Turns the party (turning on the
// spot when set) and pushes it some cells, count times.
func spinnerEvent() {
	if eventData() > 0 {
		showMazeMessage(eventData())
		waitAnimating(messageDelay)
		eraseWindow(mainWindow, true)
	}
	pause := event288()
	bits := eventBits()
	var dirs [4]int
	for i := range dirs {
		dirs[i] = extractBits(bits, i+8, 1)
	}
	count := extractBits(bits, 0, 8)
	dist := eventParam()
	turn := extractBits(bits, 12, 1)
	step := -1
	if extractBits(bits, 13, 1) == 1 {
		step = 1
	}
	randomDist := extractBits(bits, 14, 1)
	for range count {
		d := randomOpenDir(&dirs)
		if turn == 1 && d != int(partyFacing) {
			var n int
			if step == 1 {
				n = (d - int(partyFacing) + dirCount) % dirCount
			} else {
				n = (int(partyFacing) - d + dirCount) % dirCount
			}
			if dist == 0 {
				n--
			}
			for range n {
				waitAnimating(pause)
				partyFacing = int16((int(partyFacing) + step + dirCount) % dirCount)
				redrawMazeView()
			}
		}
		waitAnimating(pause)
		partyFacing = int16(d)
		if dist > 0 {
			s := &saveState
			s.SavedPartyX, s.SavedPartyY, s.SavedCellX, s.SavedCellY, s.SavedBlock, s.SavedFacing = partyX, partyY, cellX, cellY, partyBlock, partyFacing
			if randomDist == 1 {
				stepPosition(int(partyFacing), int16(random(dist)+1))
			} else {
				stepPosition(int(partyFacing), int16(dist))
			}
			if moveToMazeCell(&cellX, &cellY, &partyBlock, int(partyX), int(partyY)) == 1 {
				killParty(2022)
				return
			}
			cellEventIndex = int16(mazeB.FirstEvent[partyBlock])
			mzNextEvent = 0
		}
		redrawMazeView()
	}
	waitAnimating(pause)
}

// set_event_flags (WMAZE a1bc) sets the event's level flag: in the game (1) and also in
// the scenario header (2).
func setEventFlags(how int) {
	bit := int(mazeB.FlagBit[cellEventIndex])
	if how <= 0 || bit == 0 {
		return
	}
	setBit(levelFlags[gameStateArg][:], bit, true)
	if how > 1 {
		setBit(scenarioHdr.LevelFlags[gameStateArg][:], bit, true)
	}
}

// teleport_party (WMAZE a22c) moves the party to the event's cell, on this map or on map
// Param (99 for none, which leaves the maze).
func teleportParty() {
	bits := eventBits()
	setEventFlags(extractBits(bits, 10, 2))
	cellX = int16(extractBits(bits, 0, 3))
	cellY = int16(extractBits(bits, 3, 3))
	partyBlock = int16(extractBits(bits, 6, 4))
	m := eventParam()
	if m == 99 {
		m = -1
	}
	partyX = int16(mazeA.BlockX[partyBlock]) + cellX
	partyY = int16(mazeA.BlockY[partyBlock]) + cellY
	cellEventIndex = int16(mazeB.FirstEvent[partyBlock])
	mzNextEvent = 0
	if m == gameStateArg {
		redrawMazeView()
		return
	}
	saveLevelFlags()
	loadMazeLevel(m)
	if gameStateArg >= 0 {
		redrawMazeView()
	}
	mzEnteredCell = 1
}

// do_teleport_event (WMAZE a311).
func doTeleportEvent() {
	switch eventData() {
	case 0, 1:
		teleportParty()
	case 2:
		showMazeMessage(2052)
		playSound(8)
		waitAnimating(messageDelay / 4)
		mzClearMessage = 1
		teleportParty()
	case 4, 5, 6:
		playSound(8)
		teleportParty()
	}
}

// noopEvent18 (WMAZE a385) does nothing.
func noopEvent18() {}

// do_message_event (WMAZE a386) shows the event's text (only facing one way, if set),
// then may hand over to the treasure overlay (state 21).
func doMessageEvent() {
	bits := eventBits()
	if extractBits(bits, 14, 1) == 1 {
		eventFacing = int16(extractBits(bits, 12, 2))
		if eventFacing != partyFacing {
			return
		}
	}
	if eventData() > 0 {
		showMessageText(eventData(), 0)
	}
	if extractBits(bits, 8, 1) == 1 {
		return
	}
	treasureID = int16(eventParam())
	gameState = stateTreasureFound
}

// show_member_notice (WMAZE a421) says member m noticed something (message 2040).
func showMemberNotice(m int) {
	openEventTextWin(false)
	refreshAllWindows()
	s := replaceCharWithString(getMessage(2040), memberName(m), '^')
	setCursor(textWin, 20-(len(s)+1)/2, 1)
	printString(textWin, s, 5)
	playSound(soundBeep)
	waitAnimating(messageDelay)
	closeEventTextWin()
}

// check_secret_detect (WMAZE a4f5) lets the members, in random order, try to notice a
// secret; the first who does says so.
func checkSecretDetect() {
	if partyCounters[2].Duration > 0 && random(100) < int(partyCounters[2].B) {
		secretSpotted = 1
	}
	var tried [6]bool
	for range partyCount {
		var m int
		for {
			m = random(partyCount)
			if !tried[m] {
				break
			}
		}
		tried[m] = true
		p := &party[m]
		if p.Condition != conditionOK || p.Status[statusAsleep] != 0 || p.Status[3] != 0 {
			continue
		}
		if random(gameStateArg*10+110) < int(p.Skills[skillNoticeSecret]) {
			showMemberNotice(m)
			return
		}
	}
}

// setDirValuesEvent (WMAZE a5cd): sets the per-direction values of the current event; a
// secret among them gets a detection roll.
func setDirValuesEvent() {
	if eventData() > 0 {
		showMessageText(eventData(), 0)
	}
	secret := false
	for i := range 4 {
		curEvent[i] = int16(extractBits(eventBits(), i<<1, 2))
		if extractBits(eventBits(), i<<1+8, 2) == 2 {
			secret = true
		}
	}
	if secret {
		checkSecretDetect()
	}
	setEventFlags(extractBits(eventParam(), 5, 2))
}

// noopEvent17 (WMAZE a68d) does nothing.
func noopEvent17() {}

// dirField (WMAZE a68e) is the 2-bit field i of v.
func dirField(v, i int) int { return extractBits(v, i<<1, 2) }

// change_walls_event (WMAZE a6a8) sets the four walls of a cell from Param.
func changeWallsEvent() {
	bits := eventBits()
	cx, cy, blk := int16(extractBits(bits, 0, 3)), int16(extractBits(bits, 3, 3)), int16(extractBits(bits, 6, 4))
	redraw := extractBits(bits, 10, 1)
	flags := extractBits(bits, 11, 2)
	x := int(mazeA.BlockX[blk]) + int(cx)
	y := int(mazeA.BlockY[blk]) + int(cy)
	w := eventParam()
	cell := int(blk)<<6 + int(cy)<<3 + int(cx)
	setBitfield(mazeA.Walls0[:], cell, 2, dirField(w, 0))
	setBitfield(mazeA.Walls1[:], cell, 2, dirField(w, 1))
	if moveToMazeCell(&cx, &cy, &blk, x, y-1) == 0 {
		cell = int(blk)<<6 + int(cy)<<3 + int(cx)
		setBitfield(mazeA.Walls0[:], cell, 2, dirField(w, 2))
	}
	if moveToMazeCell(&cx, &cy, &blk, x-1, y) == 0 {
		cell = int(blk)<<6 + int(cy)<<3 + int(cx)
		setBitfield(mazeA.Walls1[:], cell, 2, dirField(w, 3))
	}
	if redraw == 1 {
		redrawMazeView()
	}
	if eventData() > 0 {
		playSound(eventData())
	}
	setEventFlags(flags)
}

// set_flag_event (WMAZE a88c): when level flags Data and Unk288 both equal the event's
// condition bit, clears, sets or toggles flag bit of map m (and copies it to the header).
func setFlagEvent() {
	if eventParam() > 0 {
		playSound(eventParam())
	}
	bits := eventBits()
	want := extractBits(bits, 15, 1) == 1
	if testBit(levelFlags[gameStateArg][:], eventData()) != want ||
		testBit(levelFlags[gameStateArg][:], event288()) != want {
		return
	}
	bit := extractBits(bits, 0, 8)
	m := extractBits(bits, 8, 4)
	switch extractBits(bits, 12, 2) {
	case 0:
		setBit(levelFlags[m][:], bit, false)
	case 1:
		setBit(levelFlags[m][:], bit, true)
	case 2:
		setBit(levelFlags[m][:], bit, !testBit(levelFlags[m][:], bit))
	}
	if extractBits(bits, 14, 1) == 1 {
		setBit(scenarioHdr.LevelFlags[m][:], bit, testBit(levelFlags[m][:], bit))
	}
}

// print_member_message (WMAZE aa50) prints member m's name and up to three words, space
// separated and centred, on row 1 of the text window.
func printMemberMessage(m int, s1, s2, s3 string, c0, c1, c2, c3 int) {
	clearWindow(textWin, false)
	refreshAllWindows()
	name := memberName(m)
	gaps := 1
	for _, s := range []string{s1, s2, s3} {
		if s != "" {
			gaps++
		}
	}
	setCursor(textWin, 20-(len(name)+len(s1)+len(s2)+len(s3)+gaps)/2, 1)
	printString(textWin, name, c0)
	textWin.CurX++
	for i, s := range []string{s1, s2, s3} {
		if s != "" {
			printString(textWin, s, []int{c1, c2, c3}[i])
			textWin.CurX++
		}
	}
}

// inflict_condition (WMAZE ab6c) worsens member m's condition to cond unless resistance
// resist saves (0: no save) and says msg; 3 and worse kill.
func inflictCondition(m, msg, resist, cond int) {
	p := &party[m]
	if (resist == 0 || random(100) > int(p.Resist[resist])) && int(p.Condition) < cond {
		p.Condition = uint8(cond)
		if cond >= 3 {
			p.HP = 0
			p.Stamina = 0
			playSound(int(p.Sex) + soundDeathBase)
		}
		printMemberMessage(m, getMessage(msg), "", "", 1, 6, 0, 0)
		drawMemberStatus(m)
		waitAnimating(messageDelay)
	}
}

// trap_damage_member (WMAZE ac20) hits member m for count dice of sides.
func trapDamageMember(m, count, sides int) {
	n := rollDice(formatDice(count, sides, 0))
	p := &party[m]
	if n == 0 || p.Condition >= conditionDead {
		return
	}
	num := itoa(n)
	pre, post := getMessage(5231), getMessage(5232)
	if int(p.HP) < n {
		n = int(p.HP)
	}
	p.HP -= int16(n)
	printMemberMessage(m, pre, num, post, 1, 5, 5, 5)
	drawMemberStatus(m)
	if p.HP <= 0 {
		waitAnimating(2)
		inflictCondition(m, 5264, 0, 3)
	} else {
		playSound(2)
		waitAnimating(messageDelay)
	}
}

// inflict_status (WMAZE ad18) adds count dice of sides plus base to member m's status
// timer status (base statusForever: for ever) unless resistance resist saves; says msg.
func inflictStatus(m, msg, resist, status, count, sides, base int) {
	p := &party[m]
	if p.Condition >= 2 {
		return
	}
	n := rollDice(formatDice(count, sides, base))
	if resist != 0 && random(100) <= int(int16(int(p.Resist[resist])-count*3)) { // C int wraps
		return
	}
	if base == statusForever {
		p.Status[status] = statusForever
	} else {
		if int16(int(p.Status[status])+n) > 250 { // C int sum wraps
			n = 250 - int(p.Status[status])
		}
		p.Status[status] += uint8(n)
	}
	if status == statusPoisoned {
		p.PoisonStr = uint8(count)
	}
	printMemberMessage(m, getMessage(msg), "", "", 1, 6, 0, 0)
	drawMemberStatus(m)
	waitAnimating(messageDelay)
}

// trap_event (WMAZE ae49): a trap; each member who fails to dodge takes damage and a
// condition or status. Kind 4 kills the party.
func trapEvent() {
	bits := eventBits()
	if extractBits(bits, 8, 4) == 4 {
		killParty(10000)
		return
	}
	eventScreenDirty = 1
	if snd := eventParam(); snd > 0 {
		if snd >= loadedSound {
			if snd != curSound {
				loadSoundBank(snd)
			}
			playSound(loadedSound)
		} else {
			playSound(snd)
		}
	}
	if eventData() > 0 {
		showMessageText(eventData(), 0)
	}
	openEventTextWin(false)
	kind := extractBits(bits, 8, 4)
	sides := extractBits(bits, 0, 4)
	count := extractBits(bits, 4, 4)
	warded := extractBits(bits, 12, 1)
	noDamage := extractBits(bits, 15, 1)
	if warded != 0 && partyCounters[5].Duration > 0 {
		noDamage = 1
		s := getMessage(2056)
		clearWindow(textWin, false)
		refreshAllWindows()
		setCursor(textWin, 20-(len(s)+1)/2, 1)
		printString(textWin, s, 6)
		playSound(soundBeep)
		waitAnimating(messageDelay)
	}
	resist := event288()
	for m := range partyCount {
		p := &party[m]
		if random(54) <= int(p.Stats[statDexterity])-count {
			continue
		}
		if noDamage == 0 && (resist == 0 || random(100) > int(p.Resist[resist])-count*3) {
			trapDamageMember(m, count, sides)
		}
		switch {
		case kind <= 0:
		case kind < 6:
			inflictCondition(m, kind+5261, resist, kind)
		default:
			inflictStatus(m, kind+5244, resist, kind-6, count, 10, count)
		}
	}
	setEventFlags(extractBits(eventBits(), 13, 2))
	closeEventTextWin()
}

// textEvent (WMAZE b123): a text event, only facing one way if set.
func textEvent() {
	bits := eventBits()
	if extractBits(bits, 6, 1) == 1 {
		eventFacing = int16(extractBits(bits, 4, 2))
		if eventFacing != partyFacing {
			return
		}
	}
	showMessageText(eventData(), extractBits(eventBits(), 2, 1))
	setEventFlags(extractBits(eventBits(), 0, 2))
}

// encounter_event (WMAZE b1c6) starts a fixed fight (state 10) with monster Param, or with
// one of encounter table Param.
func encounterEvent() {
	if extractBits(eventBits(), 3, 1) == 1 {
		encounterID = randomEncounterMonster(eventParam())
	} else {
		encounterID = eventParam()
	}
	g4338, noEscape = 1, 1
	gameState = stateBeginCombat
	setEventFlags(extractBits(eventBits(), 4, 2))
	cellEventIndex++
}

// npc_event (WMAZE b252) meets NPC Param (state 18), only facing one way if set.
func npcEvent() {
	bits := eventBits()
	if extractBits(bits, 2, 1) == 1 {
		eventFacing = int16(extractBits(bits, 0, 2))
		if eventFacing != partyFacing {
			return
		}
	}
	encounterID = eventParam()
	g4338 = 1
	gameState = stateNPC
	cellEventIndex++
}

// set_door_feature (WMAZE b2c6) closes (feature 7 with facing dir) or opens the door of
// map cell (x, y); force overrides what is there.
func setDoorFeature(x, y, dir, closed, force int) {
	var cx, cy, blk int16
	if moveToMazeCell(&cx, &cy, &blk, x, y) != 0 {
		return
	}
	cell := int(blk)<<6 + int(cy)<<3 + int(cx)
	if closed == 0 {
		if force == 1 || getBitfield(mazeA.Floor[:], cell, 4) == 7 {
			setBitfield(mazeA.Floor[:], cell, 4, 0)
			setBitfield(mazeA.Feature[:], cell, 2, 0)
		}
		return
	}
	if force != 1 && getBitfield(mazeA.Floor[:], cell, 4) != 0 {
		return
	}
	setBitfield(mazeA.Floor[:], cell, 4, 7)
	setBitfield(mazeA.Feature[:], cell, 2, dir)
}

// animate_door (WMAZE b3ac) slides the door picture (slot 2) at view depth depth, column
// col, open (1) or shut, over seven frames.
func animateDoor(opening, col, depth, sound int) {
	if dtLog != nil { // stubbed in the difftest
		return
	}
	start, step := doorStart[depth], doorStep[depth]
	if opening == 1 {
		start, step = 0, -doorStep[depth]
	}
	for i := range 7 {
		animPhase = (animPhase + 1) % 2
		drawMazeView()
		frame := depth + 10
		if depth > 0 || col == 1 {
			pic := &monsterPics[2]
			x0 := viewObjXOff[depth][col]
			drv.DrawFrames(2, x0+int(pic.FrameX[frame][0])+i*step-start, int(pic.FrameY[frame]),
				x0+doorClipL[depth], x0+doorClipR[depth], 0, false, []byte{byte(frame)})
		}
		drvShowWindow(pictureWin)
		if sound > 0 {
			playSound(sound)
		}
	}
}

// shake_view (WMAZE b500) shakes the view n times with sound.
func shakeView(n, sound int) {
	if dtLog != nil { // stubbed in the difftest
		return
	}
	for range n {
		if sound > 0 {
			animPhase = (animPhase + 1) % 2
			redrawMazeView()
			playSound(sound)
			delayLong()
		}
	}
}
