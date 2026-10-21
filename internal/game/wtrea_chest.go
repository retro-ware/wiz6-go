package game

import "strings"

// WTREA: the chest menu, its traps and the spells cast on it.

// WTREA chest state.
var (
	chestTrap         int      // chest_trap: the trap (0 none, 1..15)
	trGuessKnown      [20]int  // g5194: guessed letter i is right
	trapGuessLetters  [20]byte // trap_guess_letters
	trapGuessLen      int      // trap_guess_len
	spellDisarmResult int      // spell_disarm_result: 1 opened, -1 sprung, 0 neither
	trSpellTarget     int      // g51d4: the member a spell is cast on
)

// show_trap_effect (WTREA 5edf) says the trap went off, names it and starts trap animation
// anim (-1 none) aimed at member.
func showTrapEffect(member, anim int) {
	trLidFrame = 0
	if anim >= 0 {
		trTrapFrame = 0
		trTrapMember = member
		trTrapAnim = anim
	}
	showCenteredLine(partyPictureWin, trRow, getMessage(5227), false, 7)
	showCenteredLine(partyPictureWin, trRow, trTrapName, false, 1)
}

// trLoseMessage tells that member m lost what (message 5266 and then what).
func trLoseMessage(m, what int) {
	trNum = getMessage(what)
	trNum2 = ""
	trPrintMemberMessage(m, getMessage(5266), trNum, trNum2, 1, 5, 5, 0)
	trWaitAnimating(messageDelay)
}

// spring_trap (WTREA 5f49) sets off trap on the party (member opened the chest).
func springTrap(trap, member int) {
	clearWindow(partyPictureWin, false)
	refreshAllWindows()
	if trap == 0 {
		showCenteredLine(partyPictureWin, trRow, getMessage(5228), true, 7)
		return
	}
	lvl := gameStateArg
	switch trap {
	case 1:
		showTrapEffect(0, 15)
		for i := 0; i < partyCount; i++ {
			trTrapDamageMember(i, lvl+1, 8)
		}
	case 2:
		showTrapEffect(0, 16)
		for i := 0; i < partyCount; i++ {
			trInflictStatus(i, 5253, 10, 3, 10, 10, 50)
		}
	case 3, 9:
		showTrapEffect(0, -1)
	case 4:
		showTrapEffect(member, 6)
		trTrapDamageMember(member, 1, 6)
		trInflictStatus(member, 5255, 0, 5, lvl+2, 10, 50)
	case 5:
		showTrapEffect(0, 2)
		for i := 0; i < partyCount; i++ {
			if random(100) <= int(party[i].Resist[2]) {
				continue
			}
			switch random(6) {
			case 0:
				trTrapDamageMember(i, lvl+1, 8)
			case 1:
				trInflictStatus(i, 5254, 2, 4, 10, 10, 50)
			case 2:
				trInflictStatus(i, 5252, 9, 2, 10, 10, 50)
			case 3:
				trInflictStatus(i, 5253, 4, 3, 10, 10, 50)
			case 4:
				trInflictCondition(i, 5260, 11, 3)
			case 5:
				trInflictCondition(i, 5263, 11, 2)
			}
		}
	case 6:
		showTrapEffect(0, 16)
		for i := 0; i < partyCount; i++ {
			p := &party[i]
			trInflictStatus(i, 5265, 11, 9, 10, 10, 50)
			if random(100) > int(p.Resist[11])+75 && p.Stats[0] > 3 && p.Condition < 3 {
				p.Stats[0]--
				trLoseMessage(i, 5267)
			} else if random(100) > int(p.Resist[11])+75 && p.Stats[3] > 3 && p.Condition < 3 {
				p.Stats[3]--
				trLoseMessage(i, 5268)
			}
		}
	case 7:
		showTrapEffect(0, 16)
	case 8:
		showTrapEffect(0, 2)
		for i := 0; i < partyCount; i++ {
			p := &party[i]
			if random(100) > int(p.Resist[11])+50 && p.Condition < 3 {
				p.HPMax -= int16(random(4) + 1)
				if p.HPMax < 1 {
					p.HPMax = 1
				}
				if p.HP > p.HPMax {
					p.HP = p.HPMax
				}
				trLoseMessage(i, 5269)
			}
		}
	case 10:
		showTrapEffect(0, 16)
		for i := 0; i < partyCount; i++ {
			trInflictCondition(i, 5263, 11, 2)
		}
	case 11:
		showTrapEffect(0, 4)
	case 12:
		phantomHand()
	case 13:
		showTrapEffect(0, 3)
		for i := 0; i < partyCount; i++ {
			trTrapDamageMember(i, lvl+1, 8)
			trInflictStatus(i, 5275, 5, 8, 10, 10, 50)
		}
	case 14:
		showTrapEffect(0, 0)
		for i := 0; i < partyCount; i++ {
			trTrapDamageMember(i, lvl+1, 10)
		}
	case 15:
		showTrapEffect(0, 4)
		for i := 0; i < partyCount; i++ {
			trInflictCondition(i, 5260, 11, 3)
		}
	}
}

// phantomHand is spring_trap's trap 12: a hand beats up a random living member.
func phantomHand() {
	var i int
	for {
		if i = random(partyCount); party[i].Condition < 3 {
			break
		}
	}
	p := &party[i]
	showTrapEffect(i, 4)
	notice := func(msg int) {
		clearWindow(partyPictureWin, true)
		printCentered(partyPictureWin, getMessage(msg), trRow, 5)
		playSound(0)
		trWaitAnimating(messageDelay)
	}
	says := func(msg int) {
		trNum, trNum2 = "", ""
		trPrintMemberMessage(i, getMessage(msg), trNum, trNum2, 1, 6, 0, 0)
		trWaitAnimating(messageDelay)
	}
	notice(5270)
	says(5271)
	trTrapDamageMember(i, gameStateArg+1, 8)
	if p.Condition < 3 && random(21) > int(p.Stats[7]) {
		says(5272)
		trTrapDamageMember(i, gameStateArg<<1+2, 8)
		if random(21) > int(p.Stats[7]) {
			trInflictCondition(i, 5273, 11, 3)
		}
	}
	notice(5274)
}

// pick_chest_trap (WTREA 66b1) picks one of the chest's traps at random (0 none if it has
// none) and its name.
func pickChestTrap(chest *TreasureRec) {
	chestTrap = 0
	left, wraps := random(100)+1, 0
	for left > 0 && wraps < 100 {
		if chestTrap++; chestTrap > 15 {
			chestTrap = 0
			wraps++
		}
		if testBit(chest.Traps[:], chestTrap) {
			left--
		}
	}
	trTrapName = getMessage(chestTrap + 5300)
	trTrapPool = []byte(trTrapName)
}

// reveal_trap_letters (WTREA 672d) gives letters tries at the trap's name: each may confirm
// a guessed letter, mark it wrong, or add a letter, right with chance accuracy; then shows
// the guess in win.
func revealTrapLetters(win *Window, member, letters, accuracy int) {
	for i := 0; i < trapGuessLen && letters > 0; i++ {
		if trGuessKnown[i] != 0 || random(100) <= 50 {
			continue
		}
		if random(100) < accuracy {
			if k := strings.IndexByte(string(trTrapPool), trapGuessLetters[i]); k >= 0 {
				trGuessKnown[i] = 1
				trTrapPool[k] = '$'
			} else {
				trapGuessLetters[i] = '*'
			}
		}
		letters--
	}
	for ; letters > 0; letters-- {
		unknown := 0
		for _, c := range trTrapPool {
			if c != '$' {
				unknown++
			}
		}
		if random(100) < accuracy {
			if unknown > 0 && trapGuessLen < 18 {
				var k int
				for {
					if k = random(len(trTrapPool)); trTrapPool[k] != '$' {
						break
					}
				}
				trapGuessLetters[trapGuessLen] = trTrapPool[k]
				trGuessKnown[trapGuessLen] = 1
				trapGuessLen++
				trTrapPool[k] = '$'
			}
		} else if unknown > 0 && random(100) > 50 && trapGuessLen < 18 {
			trapGuessLetters[trapGuessLen] = byte(random(26) + 'A')
			trGuessKnown[trapGuessLen] = 0
			trapGuessLen++
		}
	}
	deferWindowDraw(win)
	clearWindowArea(win, 1, 2, 18, 2)
	setCursor(win, 1, 2)
	if chestTrap == 0 {
		printString(win, trTrapName, 6)
	} else {
		for i := 0; i < trapGuessLen; i++ {
			c := 12
			if trGuessKnown[i] == 1 {
				c = 6
			}
			putChar(win, trapGuessLetters[i], c)
		}
	}
	resumeWindowDraw(win)
	refreshWindow(win)
}

// trThiefClass is whether class c is a thief (3), or a bard, ninja or ranger (13, 4, 6),
// who also handle traps well.
func trThiefClass(c uint8) (thief, half bool) {
	return c == 3, c == 13 || c == 4 || c == 6
}

// inspect_chest (WTREA 6919): member looks at the trap, by skill (Skills[15]) and class.
func inspectChest(win *Window, member int) {
	p := &party[member]
	lv := min(int(p.Level)/5, 6)
	sk := int(p.Skills[15])
	var letters, accuracy int
	switch thief, half := trThiefClass(p.Class); {
	case thief:
		letters = sk/10 + lv - gameStateArg + 5
		accuracy = sk/2 + lv + 50 - gameStateArg<<1
	case half:
		letters = sk/10 + lv - gameStateArg + 3
		accuracy = sk/2 + lv + 35 - gameStateArg*3/2
	default:
		letters = sk/10 + lv - gameStateArg + 1
		accuracy = sk/2 + lv + 15 - gameStateArg
	}
	letters = max(letters, 1)
	accuracy = min(max(accuracy, 5), 95)
	revealTrapLetters(win, member, letters, accuracy)
}

// draw_trap_list (WTREA 6a40) shows the trap names of list around at, the one at at
// highlighted when hi.
func drawTrapList(win *Window, at int, hi bool, n int, list []int) {
	drawChestListBox(win, false)
	deferWindowDraw(win)
	for d := -2; d <= 2; d++ {
		if at+d < 0 || at+d >= n {
			continue
		}
		setCursor(win, 2, d+2)
		c := 18
		if hi && d == 0 {
			c = 6
		}
		printMessage(list[at+d]+5300, win, c)
		if hi && d == 0 {
			for win.CurX < 17 {
				putChar(win, ' ', 6)
			}
		}
	}
	resumeWindowDraw(win)
	refreshWindow(win)
}

// disarm_chest (WTREA 6ae6): member names a trap from the chest's list and tries to disarm
// it; 1 disarmed, 0 failed safely, -1 sprung.
func disarmChest(member int, chest *TreasureRec) int {
	if chestTrap == 0 {
		return -1
	}
	win := createWindow(20, 19, 20, 5, 25, -5, 0, false)
	var traps []int
	for t := range 16 {
		if testBit(chest.Traps[:], t) {
			traps = append(traps, t)
		}
	}
	sel := 0
	setClickRegion(0, win.X+2, win.Y+2, win.X+19, win.Y+2)
	setClickRegion(1, win.X+1, win.Y, win.X+1, win.Y)
	setClickRegion(2, win.X+1, win.Y+4, win.X+1, win.Y+4)
	setClickRegion(3, 255, 0, 0, 0)
	for {
		drawTrapList(win, sel, true, len(traps), traps)
		k := trInput()
		if menuClicked == 1 {
			switch k {
			case 0:
				k = 5
			case 1:
				k = 2
			case 2:
				k = 4
			}
		}
		switch k {
		case 2:
			if sel > 0 {
				sel--
			}
		case 4:
			if sel+1 < len(traps) {
				sel++
			}
		}
		if k == 5 {
			break
		}
	}
	destroyWindow(win, false)
	if traps[sel] != chestTrap {
		return -1
	}
	p := &party[member]
	chance := int(p.Skills[15])/2 + int(p.Stats[4])/2 + int(p.Level) - gameStateArg<<1
	switch thief, half := trThiefClass(p.Class); {
	case thief:
		chance += int(p.Stats[4]) * 3
		trySkillGain(member, 15, 3)
	case half:
		chance += int(p.Stats[4]) << 1
		trySkillGain(member, 15, 2)
	}
	chance = max(min(chance, 95), 5)
	return trDisarmRoll(chance)
}

// trDisarmRoll rolls against chance: 1 success, -1 (a third of the failures) sprung, 0.
func trDisarmRoll(chance int) int {
	r := random(100)
	switch {
	case r < chance:
		return 1
	case 100-(100-chance)/3 < r:
		return -1
	}
	return 0
}

// try_spell_disarm (WTREA 6d83) is the open-chest spell at power level: it may open the
// chest, spring the trap or do nothing.
func trySpellDisarm(win, trapWin *Window, member, level int) {
	for _, w := range []*Window{win, trapWin, mainWindow, bottomLineWin} {
		deferWindowDraw(w)
	}
	clearWindow(partyPictureWin, false)
	refreshAllWindows()
	chance := min(max(level*10+50-gameStateArg*5, 5), 95)
	r := random(100)
	switch {
	case r < chance:
		chestTrap = 0
		spellDisarmResult = 1
		trChestOpen = 1
		trLine = getMessage(5234)
		showCenteredLine(partyPictureWin, trRow, trLine, true, 7)
	case 100-(100-chance)/3 < r:
		spellDisarmResult = -1
	default:
		spellDisarmResult = 0
		trLine = getMessage(5235)
		showCenteredLine(partyPictureWin, trRow, trLine, true, 7)
	}
	for _, w := range []*Window{win, trapWin, mainWindow, bottomLineWin} {
		resumeWindowDraw(w)
	}
}

// cast_chest_spell (WTREA 6efd) works spell at power level by member on target (or the
// party); the chest window is win, the trap window trapWin.
func castChestSpell(member, spell, level, target int, win, trapWin *Window) {
	c := &partyCounters
	t := &party[target]
	switch spell {
	case 53:
		c[2].A += int16(level * 50)
		c[2].B = int16(clampToward(level*5+70, int(c[2].B), 0))
	case 68:
		c[3].A += int16(level * 35)
		c[3].B = int16(clampToward((level-1)/2+1, int(c[3].B), 0))
	case 41:
		c[1].A += int16(level * 50)
		c[1].B = int16(clampToward((level-1)/2+1, int(c[1].B), 0))
	case 30:
		c[5].A += int16(level * 5)
	case 38:
		c[0].A += int16(100 * level)
	case 70:
		c[4].A += int16(level * 50)
		c[4].B = int16(clampToward(level*5, int(c[4].B), 0))
	case 39:
		trySpellDisarm(win, trapWin, member, level)
	case 52:
		lv := min(int(party[member].Level), 12)
		revealTrapLetters(trapWin, member, level+4, min(level*5+65-gameStateArg<<1+lv, 95))
	case 10:
		t.Stamina = min(t.Stamina+int16(rollDice(formatDice(level<<1, 8, level))), t.StaminaMax)
		drawMemberStatus(target)
	case 15:
		t.Status[3] = uint8(max(int(t.Status[3])-level*25, 0))
		drawMemberStatus(target)
	case 26:
		n := int(t.PoisonStr) - level
		if n <= 0 {
			n = 0
			t.Status[5] = 0
		}
		t.PoisonStr = uint8(n)
		if n = int(t.Status[5]) - level*5; n <= 0 {
			n = 0
			t.PoisonStr = 0
		}
		t.Status[5] = uint8(n)
		drawMemberStatus(target)
	case 46:
		if t.Condition == 2 && level*3 > 21-int(t.Stats[3]) {
			t.Stats[3]--
			if random(21) > int(t.Stats[3]) {
				t.Condition = 3
				t.HP, t.Stamina = 0, 0
			} else {
				t.Condition = 0
			}
			drawMemberStatus(target)
		}
	case 57:
		t.Status[7] = uint8(max(int(t.Status[7])-level*25, 0))
		drawMemberStatus(target)
	case 51:
		for _, s := range []int{1, 2, 4, 8, 9} {
			t.Status[s] = uint8(max(int(t.Status[s])-level<<1, 0))
		}
		drawMemberStatus(target)
	case 64:
		t.HP = min(t.HP+int16(rollDice(formatDice(level, 7, level))), t.HPMax)
		drawMemberStatus(target)
	case 77:
		if t.Condition == 3 && level*3 > 18-int(t.Stats[3]) {
			t.Stats[3]--
			t.Condition = 0
			t.HP = 1
			t.Stamina = t.StaminaMax / 10
			t.Unk026++
			drawMemberStatus(target)
		}
	case 80:
		p := &party[member]
		p.Stamina = min(p.Stamina+int16(rollDice(formatDice(level, 8, level))), p.StaminaMax)
		p.HP = min(p.HP+int16(rollDice(formatDice(level, 2, 0))), p.HPMax)
		drawMemberStatus(member)
	case 81:
		p := &party[member]
		for r := range 6 {
			p.SP[r].Cur = min(p.SP[r].Cur+int16(rollDice(formatDice(level, 10, 0))), p.SP[r].Max)
		}
		drawMemberStatus(member)
	}
}

// unhighlight_realm_slot (WTREA 74f6) redraws realm r's glyph plain (-1: the cancel glyph).
func unhighlightRealmSlot(w *Window, r int) {
	if r == -1 {
		putGlyphAt(w, 13, 3, 34)
		return
	}
	putGlyphAt(w, r/3*6+1, r%3<<1+1, byte(r+35))
}

// draw_chest_list_box (WTREA 7549) clears a list window and draws its scroll bar.
func drawChestListBox(w *Window, refresh bool) {
	fillWindow(w, ' ', 3, false)
	deferWindowDraw(w)
	putGlyphAt(w, 1, 0, 69)
	for y := 1; y < 4; y++ {
		putGlyphAt(w, 1, y, 71)
	}
	putGlyphAt(w, 1, 4, 70)
	resumeWindowDraw(w)
	if refresh {
		refreshWindow(w)
	}
}

// draw_chest_spell_frame (WTREA 75ed) puts the frame glyphs into the window's cells.
func drawChestSpellFrame(w *Window) {
	for _, f := range spellFrame {
		end, vertical := int(f[3]%128), f[3]/128 != 0
		switch {
		case end == 0:
			wpcvwPutFrameChar(w, int(f[0]), int(f[1]), f[2])
		case !vertical:
			for x := int(f[0]); x < end; x++ {
				wpcvwPutFrameChar(w, x, int(f[1]), f[2])
			}
		default:
			for y := int(f[1]); y < end; y++ {
				wpcvwPutFrameChar(w, int(f[0]), y, f[2])
			}
		}
	}
}

// draw_chest_spell_window (WTREA 76c0) draws the realm picker with member's spell points.
func drawChestSpellWindow(w *Window, member int) {
	deferWindowDraw(w)
	fillWindow(w, 0, 1, false)
	drawChestSpellFrame(w)
	for r := range 6 {
		putGlyphAt(w, r/3*6+1, r%3<<1+1, byte(r+35))
	}
	putGlyphAt(w, 13, 3, 34)
	for r := range 6 {
		x := 3
		if r >= 3 {
			x = 9
		}
		setCursor(w, x, r%3<<1+1)
		printInt(w, int(party[member].SP[r].Cur), 3, wpcvwRealmColors[r])
	}
	for i := range 6 {
		putCharAt(w, i+15, 3, byte(i+24))
	}
	setCursor(w, 13, 5)
	printMessage(3957, w, 18)
	setCursor(w, 13, 6)
	for range 4 {
		putCharHi(w, ' ', 3)
	}
	resumeWindowDraw(w)
	refreshWindow(w)
}

// draw_chest_spell_list (WTREA 78aa) is draw_trap_list for the spells of realm r.
func drawChestSpellList(w *Window, r, at int, hi bool, n int, list []int) {
	drawChestListBox(w, false)
	deferWindowDraw(w)
	for d := -2; d <= 2; d++ {
		if at+d < 0 || at+d >= n {
			continue
		}
		setCursor(w, 2, d+2)
		c := 18
		if hi && d == 0 {
			c = wpcvwRealmColors[r]
		}
		printMessage(list[at+d]+4000, w, c)
		if hi && d == 0 {
			for w.CurX < 17 {
				putChar(w, ' ', wpcvwRealmColors[r])
			}
		}
	}
	resumeWindowDraw(w)
	refreshWindow(w)
}

// trSetBlink sets the blink spot: phase 0 shows a in colour ca, phase 1 b in colour cb.
func trSetBlink(x, y int, a string, ca int, b string, cb int) {
	trBlinkX, trBlinkY = x, y
	trBlinkText = [2]string{a, b}
	trBlinkColor = [2]int{ca, cb}
}

// choose_chest_spell (WTREA 796a) lets member pick a realm, a known spell and a power level
// (and with a targeted spell the target, trSpellTarget); level<<8 | spell, or -1.
func chooseChestSpell(win *Window, member int) int {
	p := &party[member]
	book := createWindow(0, 18, 23, 7, 25, -1, 0, false)
	list := createWindow(22, 19, 18, 6, 26, -5, 0, false)
	closeAll := func() {
		trBlink = -1
		destroyWindow(book, false)
		destroyWindow(list, false)
	}
	drawChestSpellWindow(book, member)
	drawChestListBox(list, true)
	realm := 0
	for r := range 6 {
		x, y := r/3*6+1, r%3<<1+1
		setClickRegion(r, book.X+byte(x), book.Y+byte(y), book.X+byte(x), book.Y+byte(y))
	}
	setClickRegion(6, book.X+13, book.Y+3, book.X+13, book.Y+3)
	var spells [20]int
	var count, spell, level int
	for {
		// The realm.
		setClickRegion(7, 255, 0, 0, 0)
		redraw := true
		trBlinkWin = book
		for {
			if redraw {
				trBlink = 1
				clearWindowArea(book, 13, 1, 20, 1)
				setCursor(book, 13, 1)
				if realm == -1 {
					trBlinkX, trBlinkY = 13, 3
					printMessage(3956, book, 9)
					clearWindowArea(book, 18, 5, 20, 5)
					count = -1
					drawChestListBox(list, true)
				} else {
					trBlinkX, trBlinkY = realm/3*6+1, realm%3<<1+1
					printMessage(realm+3950, book, wpcvwRealmColors[realm])
					clearWindowArea(book, 18, 5, 20, 5)
					count = wpcvwCountKnownSpells(member, realm, spells[:])
					drawChestSpellList(list, realm, 0, false, count, spells[:])
				}
				trBlinkText = [2]string{string(rune(realm + 35)), "c"}
				trBlinkColor = [2]int{17, 5}
				redraw = false
			}
			k := trInput()
			if menuClicked == 1 {
				switch {
				case k == realm:
					k = 5
				case k < 6:
					unhighlightRealmSlot(book, realm)
					realm, k = k, 0
				default:
					unhighlightRealmSlot(book, realm)
					realm, k, count = -1, 5, -1
				}
				redraw = true
			} else {
				switch k {
				case 1:
					if realm == -1 {
						unhighlightRealmSlot(book, realm)
						realm, redraw = 4, true
					} else if realm-3 >= 0 {
						unhighlightRealmSlot(book, realm)
						realm, redraw = realm-3, true
					}
				case 2:
					if realm != -1 && realm%3 > 0 {
						unhighlightRealmSlot(book, realm)
						realm, redraw = realm-1, true
					}
				case 3:
					if realm != -1 {
						unhighlightRealmSlot(book, realm)
						redraw = true
						if realm += 3; realm >= 6 {
							realm = -1
						}
					}
				case 4:
					if realm != -1 && realm%3+1 < 3 {
						unhighlightRealmSlot(book, realm)
						realm, redraw = realm+1, true
					}
				}
			}
			if k == 5 && count != 0 {
				break
			}
		}
		if realm == -1 {
			closeAll()
			return -1
		}
		// The spell.
		unhighlightRealmSlot(book, realm)
		pos, back := 0, false
		trBlinkWin = list
		trBlinkX, trBlinkY = 2, 2
		trBlinkText[0] = strings.Repeat(" ", 15) // 0x4ff5
		trBlinkColor[0] = 0
		setClickRegion(7, list.X+1, list.Y, list.X+1, list.Y)
		setClickRegion(8, list.X+1, list.Y+4, list.X+1, list.Y+4)
		setClickRegion(9, list.X+2, list.Y+2, list.X+19, list.Y+2)
		setClickRegion(10, 255, 0, 0, 0)
		for {
			trBlink = 1
			trBlinkText[1] = getMessage(spells[pos] + 4000)
			trBlinkColor[1] = wpcvwRealmColors[realm]
			drawChestSpellList(list, realm, pos, true, count, spells[:])
			cost := int(wpcvwSpells[spells[pos]].Cost)
			setCursor(book, 18, 5)
			if int(p.SP[realm].Cur) < cost {
				printString(book, "***", wpcvwRealmColors[realm]) // 0x5005
			} else {
				printInt(book, cost, 3, wpcvwRealmColors[realm])
			}
			k := trInput()
			if menuClicked == 1 {
				switch k {
				case 0, 1, 2, 3, 4, 5:
					realm, k = k, 1
				case 6:
					closeAll()
					return -1
				case 7:
					k = 2
				case 8:
					k = 4
				case 9:
					k = 5
				}
			}
			switch k {
			case 1:
				back = true
			case 2:
				if pos > 0 {
					pos--
				}
			case 4:
				if pos+1 < count {
					pos++
				}
			}
			if k == 5 || back {
				break
			}
		}
		if !back {
			spell = spells[pos]
			break
		}
	}
	// The power level.
	drawChestListBox(list, false)
	deferWindowDraw(list)
	setCursor(list, 2, 2)
	printMessage(spell+4000, list, 18)
	resumeWindowDraw(list)
	refreshWindow(list)
	trBlinkWin = book
	level = 1
	maxLevel := min(max(int(p.Level)-int(wpcvwSpells[spell].Level)+1, 1), 6)
	for l := range maxLevel {
		setClickRegion(l, book.X+byte(l)+15, book.Y+3, book.X+byte(l)+15, book.Y+3)
	}
	setClickRegion(maxLevel, book.X+13, book.Y+3, book.X+13, book.Y+3)
	setClickRegion(maxLevel+1, 255, 0, 0, 0)
	redraw := true
	plainLevel := func() {
		setCursor(book, level+14, 3)
		putChar(book, byte(level+23), 5)
	}
	var cost int
	for {
		cost = int(wpcvwSpells[spell].Cost) * level
		if redraw {
			redraw = false
			trBlink = 1
			if level == 0 {
				trSetBlink(13, 3, "\x22", 17, "c", 5)
				clearWindowArea(book, 13, 1, 20, 1)
				setCursor(book, 13, 1)
				printMessage(3956, book, 9)
				clearWindowArea(book, 18, 5, 20, 5)
			} else {
				trSetBlink(level+14, 3, " ", 0, string(rune(level+23)), 7)
				setCursor(book, 18, 5)
				if int(p.SP[realm].Cur) < cost {
					printString(book, "***", wpcvwRealmColors[realm]) // 0x5009
				} else {
					printInt(book, cost, 3, wpcvwRealmColors[realm])
				}
			}
		}
		k := trInput()
		if menuClicked == 1 {
			if k+1 == level {
				k = 5
			} else {
				plainLevel()
				level = 0
				if k < maxLevel {
					level = k + 1
				}
				k = 0
				if level == 0 {
					k = 5
				}
			}
			redraw = true
		} else {
			switch k {
			case 1:
				if level > 0 {
					plainLevel()
					level--
					redraw = true
				}
			case 3:
				if level == 0 {
					unhighlightRealmSlot(book, -1)
					clearWindowArea(book, 13, 1, 20, 1)
					setCursor(book, 13, 1)
					printMessage(realm+3950, book, wpcvwRealmColors[realm])
					level, redraw = 1, true
				} else if level < maxLevel {
					plainLevel()
					level++
					redraw = true
				}
			}
		}
		if k == 5 && int(p.SP[realm].Cur) >= cost {
			break
		}
	}
	closeAll()
	if level == 0 {
		return -1
	}
	if wpcvwSpells[spell].Target == 1 {
		if trSpellTarget = trChooseMember(2030, 1); trSpellTarget == -1 {
			return -1
		}
	}
	return level<<8 + spell
}

// cast_spell_on_chest (WTREA 8445): member casts a chosen spell and pays for it.
func castSpellOnChest(win, trapWin *Window, member int) {
	spell := chooseChestSpell(trapWin, member)
	trBlink = -1
	if spell == -1 {
		return
	}
	level, spell := spell/256, spell%256
	p := &party[member]
	s := wpcvwSpells[spell]
	p.SP[s.Realm].Cur -= int16(int(s.Cost) * level)
	castChestSpell(member, spell, level, trSpellTarget, win, trapWin)
	p.Stamina -= int16(random(level) + int(s.Level) + level)
	if p.Stamina < 0 {
		p.Stamina = 0
		p.Status[2] = uint8(random(6) + 6)
	}
	drawMemberStatus(member)
}

// close_chest_windows (WTREA 90d4).
func closeChestWindows(win, trapWin *Window) {
	destroyWindow(trapWin, false)
	destroyWindow(win, false)
	deferWindowDraw(mainWindow)
	deferWindowDraw(bottomLineWin)
}

// handle_chest (WTREA 9105) runs the chest menu (inspect, disarm, open, spell, leave) until
// the chest is open (0) or left (-1).
func handleChest(chest *TreasureRec) int {
	trChestPhase = 0
	switch chest.ChestType {
	case 0:
		trChestKind = -1
	case 1:
		trChestKind = 0
	case 2:
		trChestKind = 1
	}
	pickChestTrap(chest)
	resumeWindowDraw(mainWindow)
	resumeWindowDraw(bottomLineWin)
	eraseCursorMark(bottomLineWin, false)
	win := createWindow(0, 19, 19, 5, 20, -4, 0, false)
	fillWindow(win, ' ', 3, false)
	trapWin := createWindow(20, 19, 20, 5, 21, -5, 0, false)
	fillWindow(trapWin, ' ', 3, false)
	deferWindowDraw(trapWin)
	setCursor(trapWin, 4, 0)
	printMessage(5226, trapWin, 18)
	g := func(x, y int, ch byte) { wpcvwPutCharC1(trapWin, x, y, ch) }
	g(0, 1, 17)
	g(19, 1, 19)
	g(0, 3, 6)
	g(19, 3, 8)
	for x := 1; x < 19; x++ {
		g(x, 1, 18)
		g(x, 3, 7)
	}
	g(0, 2, 4)
	g(19, 2, 5)
	clearWindowArea(trapWin, 1, 2, 18, 2)
	resumeWindowDraw(trapWin)
	trapGuessLen = 0
	trapGuessLetters, trGuessKnown = [20]byte{}, [20]int{}
	var inspected [6]bool
	options := []int{1, 1, 1, 1, 1, -1} // tried[6..10] and l22 behind it
	for {
		printChestPrompt(5200, 2)
		fillWindow(win, ' ', 3, false)
		refreshAllWindows()
		switch chestMenuSelect(win, 5201, options, 2, 1, 10, 3, 5, 0) {
		case 0: // inspect
			if m := trChooseMember(5223, 0); m != -1 {
				if inspected[m] {
					playSound(0)
				} else if isMemberAble(m) {
					inspectChest(trapWin, m)
					inspected[m] = true
				}
			}
		case 1: // disarm
			m := trChooseMember(5220, 0)
			if m == -1 || !isMemberAble(m) {
				break
			}
			r := disarmChest(m, chest)
			for _, w := range []*Window{win, trapWin, mainWindow, bottomLineWin} {
				deferWindowDraw(w)
			}
			clearWindow(partyPictureWin, false)
			refreshAllWindows()
			switch r {
			case 1:
				trChestOpen = 1
				showCenteredLine(partyPictureWin, trRow, getMessage(5229), true, 7)
				closeChestWindows(win, trapWin)
				return 0
			case -1:
				trChestOpen = 1
				closeChestWindows(win, trapWin)
				springTrap(chestTrap, m)
				return 0
			}
			showCenteredLine(partyPictureWin, trRow, getMessage(5230), true, 7)
			for _, w := range []*Window{win, trapWin, mainWindow, bottomLineWin} {
				resumeWindowDraw(w)
			}
		case 2: // open
			if m := trChooseMember(5225, 0); m != -1 && isMemberAble(m) {
				trChestOpen = 1
				closeChestWindows(win, trapWin)
				springTrap(chestTrap, m)
				return 0
			}
		case 3: // spell
			m := trChooseMember(5224, 0)
			if m == -1 || !isMemberAble(m) {
				break
			}
			spellDisarmResult = 0
			castSpellOnChest(win, trapWin, m)
			switch spellDisarmResult {
			case 1:
				trChestOpen = 1
				closeChestWindows(win, trapWin)
				return 0
			case -1:
				trChestOpen = 1
				closeChestWindows(win, trapWin)
				springTrap(chestTrap, m)
				return 0
			}
		case 4: // leave
			closeChestWindows(win, trapWin)
			return -1
		}
	}
}
