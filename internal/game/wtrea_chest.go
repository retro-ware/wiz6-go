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

const (
	trapCount         = 16   // chest traps 0 (none) .. 15, TreasureRec.Traps bits
	msgTrapNameBase   = 5300 // plus the trap: its name
	trapGuessMax      = 18   // the letters of a trap guess
	trapLetterGuessed = '$'  // a trTrapPool letter already found
	trapLetterWrong   = '*'  // a guessed letter found wrong
	spellOpenChest    = 39   // may open the chest or spring its trap (spellLockInNPC in a meeting)
	spellRevealTrap   = 52   // reveals letters of the trap's name
	disarmOpened      = 1    // disarmChest, spellDisarmResult: the chest is open
	disarmFailed      = 0    // nothing happened
	disarmSprung      = -1   // the trap went off
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
	showCenteredLine(partyPictureWin, trRow, getMessage(msgTrapWasTriggered), false, 7)
	showCenteredLine(partyPictureWin, trRow, trTrapName, false, 1)
}

// trLoseMessage tells that member m lost what (message 5266 and then what).
func trLoseMessage(m, what int) {
	lost := getMessage(msgLost)
	trNum = getMessage(what)
	trNum2 = ""
	trPrintMemberMessage(m, lost, trNum, trNum2, 1, 5, 5, 0)
	trWaitAnimating(messageDelay)
}

// spring_trap (WTREA 5f49) sets off trap on the party (member opened the chest).
func springTrap(trap, member int) {
	clearWindow(partyPictureWin, false)
	refreshAllWindows()
	if trap == 0 {
		showCenteredLine(partyPictureWin, trRow, getMessage(msgChestWasNotTrapped), true, 7)
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
			trInflictStatus(i, msgTrapStatusBase+statusParalyzed, 10, statusParalyzed, 10, 10, 50)
		}
	case 3, 9:
		showTrapEffect(0, -1)
	case 4:
		showTrapEffect(member, 6)
		trTrapDamageMember(member, 1, 6)
		trInflictStatus(member, msgTrapStatusBase+statusPoisoned, 0, statusPoisoned, lvl+2, 10, 50)
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
				trInflictStatus(i, msgTrapStatusBase+statusBlinded, 2, statusBlinded, 10, 10, 50)
			case 2:
				trInflictStatus(i, msgTrapStatusBase+statusAsleep, 9, statusAsleep, 10, 10, 50)
			case 3:
				trInflictStatus(i, msgTrapStatusBase+statusParalyzed, 4, statusParalyzed, 10, 10, 50)
			case 4:
				trInflictCondition(i, msgSlain, 11, conditionDead)
			case 5:
				trInflictCondition(i, msgTrapConditionBase+conditionStoned, 11, conditionStoned)
			}
		}
	case 6:
		showTrapEffect(0, 16)
		for i := 0; i < partyCount; i++ {
			p := &party[i]
			trInflictStatus(i, msgWeakened, 11, 9, 10, 10, 50)
			if random(100) > int(p.Resist[11])+75 && p.Stats[statStrength] > 3 && p.Condition < conditionDead {
				p.Stats[statStrength]--
				trLoseMessage(i, msgStrength)
			} else if random(100) > int(p.Resist[11])+75 && p.Stats[statVitality] > 3 && p.Condition < conditionDead {
				p.Stats[statVitality]--
				trLoseMessage(i, msgVitality)
			}
		}
	case 7:
		showTrapEffect(0, 16)
	case 8:
		showTrapEffect(0, 2)
		for i := 0; i < partyCount; i++ {
			p := &party[i]
			if random(100) > int(p.Resist[11])+50 && p.Condition < conditionDead {
				p.HPMax -= int16(random(4) + 1)
				if p.HPMax < 1 {
					p.HPMax = 1
				}
				if p.HP > p.HPMax {
					p.HP = p.HPMax
				}
				trLoseMessage(i, msgHitPoints)
			}
		}
	case 10:
		showTrapEffect(0, 16)
		for i := 0; i < partyCount; i++ {
			trInflictCondition(i, msgTrapConditionBase+conditionStoned, 11, conditionStoned)
		}
	case 11:
		showTrapEffect(0, 4)
	case 12:
		phantomHand()
	case 13:
		showTrapEffect(0, 3)
		for i := 0; i < partyCount; i++ {
			trTrapDamageMember(i, lvl+1, 8)
			trInflictStatus(i, msgNauseated, 5, statusNauseated, 10, 10, 50)
		}
	case 14:
		showTrapEffect(0, 0)
		for i := 0; i < partyCount; i++ {
			trTrapDamageMember(i, lvl+1, 10)
		}
	case 15:
		showTrapEffect(0, 4)
		for i := 0; i < partyCount; i++ {
			trInflictCondition(i, msgSlain, 11, conditionDead)
		}
	}
}

// phantomHand is spring_trap's trap 12: a hand beats up a random living member.
func phantomHand() {
	var i int
	for {
		if i = random(partyCount); party[i].Condition < conditionDead {
			break
		}
	}
	p := &party[i]
	showTrapEffect(i, 4)
	notice := func(msg int) {
		clearWindow(partyPictureWin, true)
		printCentered(partyPictureWin, getMessage(msg), trRow, 5)
		playSound(soundBeep)
		trWaitAnimating(messageDelay)
	}
	says := func(msg int) {
		trNum, trNum2 = "", ""
		trPrintMemberMessage(i, getMessage(msg), trNum, trNum2, 1, 6, 0, 0)
		trWaitAnimating(messageDelay)
	}
	notice(msgPhantasmalHandMaterializes)
	says(msgIsAttacked)
	trTrapDamageMember(i, gameStateArg+1, 8)
	if p.Condition < conditionDead && random(21) > int(p.Stats[statKarma]) {
		says(msgIsBeatenSeverely)
		trTrapDamageMember(i, gameStateArg<<1+2, 8)
		if random(21) > int(p.Stats[statKarma]) {
			trInflictCondition(i, msgIsChokedSmashedAnd, 11, conditionDead)
		}
	}
	notice(msgPhantomHandWavesAnd)
}

// pick_chest_trap (WTREA 66b1) picks one of the chest's traps at random (0 none if it has
// none) and its name.
func pickChestTrap(chest *TreasureRec) {
	chestTrap = 0
	left, wraps := random(100)+1, 0
	for left > 0 && wraps < 100 {
		if chestTrap++; chestTrap >= trapCount {
			chestTrap = 0
			wraps++
		}
		if testBit(chest.Traps[:], chestTrap) {
			left--
		}
	}
	trTrapName = getMessage(chestTrap + msgTrapNameBase)
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
				trTrapPool[k] = trapLetterGuessed
			} else {
				trapGuessLetters[i] = trapLetterWrong
			}
		}
		letters--
	}
	for ; letters > 0; letters-- {
		unknown := 0
		for _, c := range trTrapPool {
			if c != trapLetterGuessed {
				unknown++
			}
		}
		if random(100) < accuracy {
			if unknown > 0 && trapGuessLen < trapGuessMax {
				var k int
				for {
					if k = random(len(trTrapPool)); trTrapPool[k] != trapLetterGuessed {
						break
					}
				}
				trapGuessLetters[trapGuessLen] = trTrapPool[k]
				trGuessKnown[trapGuessLen] = 1
				trapGuessLen++
				trTrapPool[k] = trapLetterGuessed
			}
		} else if unknown > 0 && random(100) > 50 && trapGuessLen < trapGuessMax {
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

// trThiefClass is whether class c is a thief, or a ninja, ranger or bard, who also
// handle traps well.
func trThiefClass(c uint8) (thief, half bool) {
	return c == classThief, c == classNinja || c == classRanger || c == classBard
}

// inspect_chest (WTREA 6919): member looks at the trap, by skill (skillPickLock) and class.
func inspectChest(win *Window, member int) {
	p := &party[member]
	lv := min(int(p.Level)/5, 6)
	sk := int(p.Skills[skillPickLock])
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
	if dtLog != nil { // the difftest stubs it
		return
	}
	drawListFrame(win, false)
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
		printMessage(list[at+d]+msgTrapNameBase, win, c)
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
		return disarmSprung
	}
	win := createWindow(20, 19, 20, 5, 25, borderTopEdge, 0, false)
	var traps []int
	for t := range trapCount {
		if testBit(chest.Traps[:], t) {
			traps = append(traps, t)
		}
	}
	sel := 0
	setClickRegion(0, win.X+2, win.Y+2, win.X+19, win.Y+2)
	setClickRegion(1, win.X+1, win.Y, win.X+1, win.Y)
	setClickRegion(2, win.X+1, win.Y+4, win.X+1, win.Y+4)
	setClickRegion(3, clickRegionsEnd, 0, 0, 0)
	for {
		drawTrapList(win, sel, true, len(traps), traps)
		k := trInput()
		if menuClicked == 1 {
			switch k {
			case 0:
				k = menuKeyEnter
			case 1:
				k = menuKeyUp
			case 2:
				k = menuKeyDown
			}
		}
		switch k {
		case menuKeyUp:
			if sel > 0 {
				sel--
			}
		case menuKeyDown:
			if sel+1 < len(traps) {
				sel++
			}
		}
		if k == menuKeyEnter {
			break
		}
	}
	destroyWindow(win, false)
	if traps[sel] != chestTrap {
		return disarmSprung
	}
	p := &party[member]
	chance := int(p.Skills[skillPickLock])/2 + int(p.Stats[statDexterity])/2 + int(p.Level) - gameStateArg<<1
	switch thief, half := trThiefClass(p.Class); {
	case thief:
		chance += int(p.Stats[statDexterity]) * 3
		trySkillGain(member, skillPickLock, 3)
	case half:
		chance += int(p.Stats[statDexterity]) << 1
		trySkillGain(member, skillPickLock, 2)
	}
	chance = max(min(chance, 95), 5)
	return trDisarmRoll(chance)
}

// trDisarmRoll rolls against chance: 1 success, -1 (a third of the failures) sprung, 0.
func trDisarmRoll(chance int) int {
	r := random(100)
	switch {
	case r < chance:
		return disarmOpened
	case 100-(100-chance)/3 < r:
		return disarmSprung
	}
	return disarmFailed
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
		spellDisarmResult = disarmOpened
		trChestOpen = 1
		trLine = getMessage(msgChestOpens)
		showCenteredLine(partyPictureWin, trRow, trLine, true, 7)
	case 100-(100-chance)/3 < r:
		spellDisarmResult = disarmSprung
	default:
		spellDisarmResult = disarmFailed
		trLine = getMessage(msgChestDidNotOpen)
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
	case spellDetectSecret:
		c[effectDetectSecret].Duration += int16(level * 50)
		c[effectDetectSecret].B = int16(clampToward(level*5+70, int(c[effectDetectSecret].B), 0))
	case spellEnchantedBlade:
		c[effectEnchantedBlade].Duration += int16(level * 35)
		c[effectEnchantedBlade].B = int16(clampToward((level-1)/2+1, int(c[effectEnchantedBlade].B), 0))
	case spellArmorplate:
		c[effectArmorplate].Duration += int16(level * 50)
		c[effectArmorplate].B = int16(clampToward((level-1)/2+1, int(c[effectArmorplate].B), 0))
	case spellLevitate:
		c[effectLevitate].Duration += int16(level * 5)
	case spellDirection:
		c[effectDirection].Duration += int16(100 * level)
	case spellMagicScreen:
		c[effectMagicScreen].Duration += int16(level * 50)
		c[effectMagicScreen].B = int16(clampToward(level*5, int(c[effectMagicScreen].B), 0))
	case spellOpenChest:
		trySpellDisarm(win, trapWin, member, level)
	case spellRevealTrap:
		lv := min(int(party[member].Level), 12)
		revealTrapLetters(trapWin, member, level+4, min(level*5+65-gameStateArg<<1+lv, 95))
	case spellStamina:
		t.Stamina = min(t.Stamina+int16(rollDice(formatDice(level<<1, 8, level))), t.StaminaMax)
		drawMemberStatus(target)
	case spellCureParalysis:
		t.Status[statusParalyzed] = uint8(max(int(int16(int(t.Status[statusParalyzed])-level*25)), 0)) // C int wraps
		drawMemberStatus(target)
	case spellCurePoison:
		n := int(int16(int(t.PoisonStr) - level)) // C int wraps
		if n <= 0 {
			n = 0
			t.Status[statusPoisoned] = 0
		}
		t.PoisonStr = uint8(n)
		if n = int(int16(int(t.Status[statusPoisoned]) - level*5)); n <= 0 { // C int wraps
			n = 0
			t.PoisonStr = 0
		}
		t.Status[statusPoisoned] = uint8(n)
		drawMemberStatus(target)
	case spellCureStone:
		if t.Condition == conditionStoned && int(int16(level*3)) > 21-int(t.Stats[statVitality]) { // C int wraps
			t.Stats[statVitality]--
			if random(21) > int(t.Stats[statVitality]) {
				t.Condition = conditionDead
				t.HP, t.Stamina = 0, 0
			} else {
				t.Condition = conditionOK
			}
			drawMemberStatus(target)
		}
	case spellSaneMind:
		t.Status[statusInsane] = uint8(max(int(int16(int(t.Status[statusInsane])-level*25)), 0)) // C int wraps
		drawMemberStatus(target)
	case spellCureLesserCnd:
		for _, s := range []int{1, statusAsleep, 4, 8, 9} {
			t.Status[s] = uint8(max(int(int16(int(t.Status[s])-level<<1)), 0)) // C int wraps
		}
		drawMemberStatus(target)
	case spellHealWounds:
		t.HP = min(t.HP+int16(rollDice(formatDice(level, 7, level))), t.HPMax)
		drawMemberStatus(target)
	case spellResurrection:
		if t.Condition == conditionDead && int(int16(level*3)) > 18-int(t.Stats[statVitality]) { // C int wraps
			t.Stats[statVitality]--
			t.Condition = conditionOK
			t.HP = 1
			t.Stamina = t.StaminaMax / 10
			t.Unk026++
			drawMemberStatus(target)
		}
	case spellHelpFood:
		p := &party[member]
		p.Stamina = min(p.Stamina+int16(rollDice(formatDice(level, 8, level))), p.StaminaMax)
		p.HP = min(p.HP+int16(rollDice(formatDice(level, 2, 0))), p.HPMax)
		drawMemberStatus(member)
	case spellMagicFood:
		p := &party[member]
		for r := range realmCount {
			p.SP[r].Cur = min(p.SP[r].Cur+int16(rollDice(formatDice(level, 10, 0))), p.SP[r].Max)
		}
		drawMemberStatus(member)
	}
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
	book := createWindow(0, 18, 23, 7, 25, borderNone, 0, false)
	list := createWindow(22, 19, 18, 6, 26, borderTopEdge, 0, false)
	closeAll := func() {
		trBlink = -1
		destroyWindow(book, false)
		destroyWindow(list, false)
	}
	drawBookSpellWindow(book, member)
	drawListFrame(list, true)
	realm := 0
	for r := range realmCount {
		x, y := r/3*6+1, r%3<<1+1
		setClickRegion(r, book.X+byte(x), book.Y+byte(y), book.X+byte(x), book.Y+byte(y))
	}
	setClickRegion(6, book.X+13, book.Y+3, book.X+13, book.Y+3)
	var spells [20]int
	var count, spell, level int
	for {
		// The realm.
		setClickRegion(7, clickRegionsEnd, 0, 0, 0)
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
					drawListFrame(list, true)
				} else {
					trBlinkX, trBlinkY = realm/3*6+1, realm%3<<1+1
					printMessage(realm+msgRealmBase, book, realmColors[realm])
					clearWindowArea(book, 18, 5, 20, 5)
					count = countKnownSpells(member, realm, spells[:])
					drawBookSpellList(list, realm, 0, false, count, spells[:])
				}
				trBlinkText = [2]string{string(rune(realm + 35)), "c"}
				trBlinkColor = [2]int{17, 5}
				redraw = false
			}
			k := trInput()
			if menuClicked == 1 {
				switch {
				case k == realm:
					k = menuKeyEnter
				case k < realmCount:
					unhighlightBookRealm(book, realm)
					realm, k = k, menuKeyEscape
				default:
					unhighlightBookRealm(book, realm)
					realm, k, count = -1, menuKeyEnter, -1
				}
				redraw = true
			} else {
				switch k {
				case menuKeyLeft:
					if realm == -1 {
						unhighlightBookRealm(book, realm)
						realm, redraw = 4, true
					} else if realm-3 >= 0 {
						unhighlightBookRealm(book, realm)
						realm, redraw = realm-3, true
					}
				case menuKeyUp:
					if realm != -1 && realm%3 > 0 {
						unhighlightBookRealm(book, realm)
						realm, redraw = realm-1, true
					}
				case menuKeyRight:
					if realm != -1 {
						unhighlightBookRealm(book, realm)
						redraw = true
						if realm += 3; realm >= realmCount {
							realm = -1
						}
					}
				case menuKeyDown:
					if realm != -1 && realm%3+1 < 3 {
						unhighlightBookRealm(book, realm)
						realm, redraw = realm+1, true
					}
				}
			}
			if k == menuKeyEnter && count != 0 {
				break
			}
		}
		if realm == -1 {
			closeAll()
			return -1
		}
		// The spell.
		unhighlightBookRealm(book, realm)
		pos, back := 0, false
		trBlinkWin = list
		trBlinkX, trBlinkY = 2, 2
		trBlinkText[0] = strings.Repeat(" ", 15) // 0x4ff5
		trBlinkColor[0] = 0
		setClickRegion(7, list.X+1, list.Y, list.X+1, list.Y)
		setClickRegion(8, list.X+1, list.Y+4, list.X+1, list.Y+4)
		setClickRegion(9, list.X+2, list.Y+2, list.X+19, list.Y+2)
		setClickRegion(10, clickRegionsEnd, 0, 0, 0)
		for {
			trBlink = 1
			trBlinkText[1] = getMessage(spells[pos] + msgSpellNameBase)
			trBlinkColor[1] = realmColors[realm]
			drawBookSpellList(list, realm, pos, true, count, spells[:])
			cost := int(spellTable[spells[pos]].Cost)
			setCursor(book, 18, 5)
			if int(p.SP[realm].Cur) < cost {
				printString(book, "***", realmColors[realm]) // 0x5005
			} else {
				printInt(book, cost, 3, realmColors[realm])
			}
			k := trInput()
			if menuClicked == 1 {
				switch k {
				case 0, 1, 2, 3, 4, 5:
					realm, k = k, menuKeyLeft
				case 6:
					closeAll()
					return -1
				case 7:
					k = menuKeyUp
				case 8:
					k = menuKeyDown
				case 9:
					k = menuKeyEnter
				}
			}
			switch k {
			case menuKeyLeft:
				back = true
			case menuKeyUp:
				if pos > 0 {
					pos--
				}
			case menuKeyDown:
				if pos+1 < count {
					pos++
				}
			}
			if k == menuKeyEnter || back {
				break
			}
		}
		if !back {
			spell = spells[pos]
			break
		}
	}
	// The power level.
	drawListFrame(list, false)
	deferWindowDraw(list)
	setCursor(list, 2, 2)
	printMessage(spell+msgSpellNameBase, list, 18)
	resumeWindowDraw(list)
	refreshWindow(list)
	trBlinkWin = book
	level = 1
	maxLevel := min(max(int(p.Level)-int(spellTable[spell].Level)+1, 1), 6)
	for l := range maxLevel {
		setClickRegion(l, book.X+byte(l)+15, book.Y+3, book.X+byte(l)+15, book.Y+3)
	}
	setClickRegion(maxLevel, book.X+13, book.Y+3, book.X+13, book.Y+3)
	setClickRegion(maxLevel+1, clickRegionsEnd, 0, 0, 0)
	redraw := true
	plainLevel := func() {
		setCursor(book, level+14, 3)
		putChar(book, byte(level+23), 5)
	}
	var cost int
	for {
		cost = int(spellTable[spell].Cost) * level
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
					printString(book, "***", realmColors[realm]) // 0x5009
				} else {
					printInt(book, cost, 3, realmColors[realm])
				}
			}
		}
		k := trInput()
		if menuClicked == 1 {
			if k+1 == level {
				k = menuKeyEnter
			} else {
				plainLevel()
				level = 0
				if k < maxLevel {
					level = k + 1
				}
				k = menuKeyEscape
				if level == 0 {
					k = menuKeyEnter
				}
			}
			redraw = true
		} else {
			switch k {
			case menuKeyLeft:
				if level > 0 {
					plainLevel()
					level--
					redraw = true
				}
			case menuKeyRight:
				if level == 0 {
					unhighlightBookRealm(book, -1)
					clearWindowArea(book, 13, 1, 20, 1)
					setCursor(book, 13, 1)
					printMessage(realm+msgRealmBase, book, realmColors[realm])
					level, redraw = 1, true
				} else if level < maxLevel {
					plainLevel()
					level++
					redraw = true
				}
			}
		}
		if k == menuKeyEnter && int(p.SP[realm].Cur) >= cost {
			break
		}
	}
	closeAll()
	if level == 0 {
		return -1
	}
	if spellTable[spell].Target == spellTargetMember {
		if trSpellTarget = trChooseMember(msgCastOnWho, 1); trSpellTarget == -1 {
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
	s := spellTable[spell]
	p.SP[s.Realm].Cur -= int16(int(s.Cost) * level)
	castChestSpell(member, spell, level, trSpellTarget, win, trapWin)
	p.Stamina -= int16(random(level) + int(s.Level) + level)
	if p.Stamina < 0 {
		p.Stamina = 0
		p.Status[statusAsleep] = uint8(random(6) + 6)
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
	win := createWindow(0, 19, 19, 5, 20, borderRightEdge, 0, false)
	fillWindow(win, ' ', 3)
	trapWin := createWindow(20, 19, 20, 5, 21, borderTopEdge, 0, false)
	fillWindow(trapWin, ' ', 3)
	deferWindowDraw(trapWin)
	setCursor(trapWin, 4, 0)
	printMessage(msgTrap, trapWin, 18)
	g := func(x, y int, ch byte) { putGlyphAtC1(trapWin, x, y, ch) }
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
		printChestPrompt(msgChest, 2)
		fillWindow(win, ' ', 3)
		refreshAllWindows()
		switch menuSelectCancel(win, msgChestMenu, options, 2, 1, 10, 3, 5, 0, 0, trInput) {
		case 0: // inspect
			if m := trChooseMember(msgWhoInspects, 0); m != -1 {
				if inspected[m] {
					playSound(soundBeep)
				} else if isMemberAble(m) {
					inspectChest(trapWin, m)
					inspected[m] = true
				}
			}
		case 1: // disarm
			m := trChooseMember(msgWhoDisarms, 0)
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
			case disarmOpened:
				trChestOpen = 1
				showCenteredLine(partyPictureWin, trRow, getMessage(msgYouDisarmedTrap), true, 7)
				closeChestWindows(win, trapWin)
				return 0
			case disarmSprung:
				trChestOpen = 1
				closeChestWindows(win, trapWin)
				springTrap(chestTrap, m)
				return 0
			}
			showCenteredLine(partyPictureWin, trRow, getMessage(msgYouCouldNotDisarm), true, 7)
			for _, w := range []*Window{win, trapWin, mainWindow, bottomLineWin} {
				resumeWindowDraw(w)
			}
		case 2: // open
			if m := trChooseMember(msgWhoOpens, 0); m != -1 && isMemberAble(m) {
				trChestOpen = 1
				closeChestWindows(win, trapWin)
				springTrap(chestTrap, m)
				return 0
			}
		case 3: // spell
			m := trChooseMember(msgChestWhoCasts, 0)
			if m == -1 || !isMemberAble(m) {
				break
			}
			spellDisarmResult = disarmFailed
			castSpellOnChest(win, trapWin, m)
			switch spellDisarmResult {
			case disarmOpened:
				trChestOpen = 1
				closeChestWindows(win, trapWin)
				return 0
			case disarmSprung:
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
