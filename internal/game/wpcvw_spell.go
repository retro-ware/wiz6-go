package game

// WPCVW: casting spells outside combat, the spell book and learning spells.

// clampToward (clamp_toward, WPCVW 6aec): mode 1 lowers v by n if positive, else keeps it
// at least -n; mode 0 raises a negative v by n, else makes it at least n.
func clampToward(n, v, mode int) int {
	switch mode {
	case 1:
		if v > 0 {
			v -= n
		} else if -n < v {
			v = -n
		}
	case 0:
		if v < 0 {
			v += n
		} else if v < n {
			v = n
		}
	}
	return v
}

// wpcvwRefreshCard (WPCVW 6ac3) shows member m's changed card for a moment.
func wpcvwRefreshCard(w *Window, m int) {
	resumeWindowDraw(w)
	drawMemberCard(w, m)
	waitSecondsOrEnter(4)
	deferWindowDraw(w)
}

// castSpell (cast_spell, WPCVW 6b3c) casts spell s at power of caster m on member t.
func castSpell(m, s, power, t int, sheet, menu *Window) {
	c := &partyCounters
	q := &party[t]
	lower := func(v *uint8, by int) { *v = uint8(max(int(*v)-by, 0)) }
	switch s {
	case 53:
		c[2].A += int16(power * 50)
		c[2].B = int16(clampToward(power*5+70, int(c[2].B), 0))
	case 68:
		c[3].A += int16(power * 35)
		c[3].B = int16(clampToward((power-1)/2+1, int(c[3].B), 0))
	case 41:
		c[1].A += int16(power * 50)
		c[1].B = int16(clampToward((power-1)/2+1, int(c[1].B), 0))
	case 30:
		c[5].A += int16(power * 5)
	case 38:
		c[0].A += int16(100 * power)
	case 70:
		c[4].A += int16(power * 50)
		c[4].B = int16(clampToward(power*5, int(c[4].B), 0))
	case 10:
		q.Stamina = min(q.Stamina+int16(rollDice(formatDice(power<<1, 8, power))), q.StaminaMax)
		wpcvwRefreshCard(pvCardWin, t)
	case 15:
		lower(&q.Status[3], power*25)
		wpcvwRefreshCard(pvCardWin, t)
	case 26:
		if int(q.PoisonStr)-power <= 0 {
			q.PoisonStr, q.Status[5] = 0, 0
		} else {
			q.PoisonStr -= uint8(power)
		}
		if int(q.Status[5])-power*5 <= 0 {
			q.Status[5], q.PoisonStr = 0, 0
		} else {
			q.Status[5] -= uint8(power * 5)
		}
		wpcvwRefreshCard(pvCardWin, t)
	case 46:
		if q.Condition == 2 && power*3 > 21-int(q.Stats[3]) {
			q.Stats[3]--
			if random(21) > int(q.Stats[3]) {
				q.Condition, q.HP, q.Stamina = 3, 0, 0
			} else {
				q.Condition = 0
			}
			wpcvwRefreshCard(pvCardWin, t)
		}
	case 57:
		lower(&q.Status[7], power*25)
		wpcvwRefreshCard(pvCardWin, t)
	case 51:
		for _, k := range []int{1, 2, 4, 8, 9} {
			lower(&q.Status[k], power<<1)
		}
		wpcvwRefreshCard(pvCardWin, t)
	case 64:
		q.HP = min(q.HP+int16(rollDice(formatDice(power, 7, power))), q.HPMax)
		wpcvwRefreshCard(pvCardWin, t)
	case 77:
		if q.Condition == 3 && power*3 > 18-int(q.Stats[3]) {
			q.Stats[3]--
			q.Condition, q.HP, q.Stamina = 0, 1, q.StaminaMax/10
			q.Unk026++
			wpcvwRefreshCard(pvCardWin, t)
		}
	case 80:
		p := &party[m]
		p.Stamina = min(p.Stamina+int16(rollDice(formatDice(power, 8, power))), p.StaminaMax)
		p.HP = min(p.HP+int16(rollDice(formatDice(power, 2, 0))), p.HPMax)
	case 81:
		p := &party[m]
		for r := range p.SP {
			p.SP[r].Cur = min(p.SP[r].Cur+int16(rollDice(formatDice(power, 10, 0))), p.SP[r].Max)
		}
	case 54:
		if party[m].ItemCount > 0 {
			refreshAllWindows()
			if pvChosenItem = chooseItem(sheet, menu, 481, m, false); pvChosenItem != -1 {
				showItemInfo(menu, m, pvChosenItem, power)
			}
		}
	case 73:
		// As in the C, removing shifts the list, so the next item is skipped.
		for i := 0; i < int(q.ItemCount); i++ {
			if q.Items[i].Flags&2 != 0 {
				wpcvwRemoveItem(t, i)
			}
		}
	}
}

// wpcvwDrawSpellListFrame (WPCVW 719f) clears the spell list window and draws its left edge.
func wpcvwDrawSpellListFrame(w *Window, refresh bool) {
	fillWindow(w, ' ', 3, true)
	deferWindowDraw(w)
	putGlyphAt(w, 0, 0, 69)
	for y := 1; y < 6; y++ {
		putGlyphAt(w, 0, y, 71)
	}
	putGlyphAt(w, 0, 6, 70)
	resumeWindowDraw(w)
	if refresh {
		refreshWindow(w)
	}
}

// wpcvwDrawSpellWindow (WPCVW 721a) draws the spell book frame: title, realm and power rows.
func wpcvwDrawSpellWindow(w *Window) {
	f := func(x, y int, ch byte) { wpcvwPutFrameChar(w, x, y, ch) }
	deferWindowDraw(w)
	fillWindow(w, 0, 1, true)
	f(0, 0, 14)
	f(19, 0, 10)
	f(0, 2, 21)
	f(19, 2, 8)
	for x := 1; x < 19; x++ {
		f(x, 0, 12)
		f(x, 2, 7)
	}
	f(0, 1, 13)
	f(19, 1, 5)
	setCursor(w, 1, 1)
	printMessage(700, w, 18)
	wpcvwPutIconAt(w, 2, 1, 34)
	wpcvwPutIconAt(w, 17, 1, 34)
	for y := 3; y < 9; y++ {
		f(0, y, 5)
	}
	for _, c := range [][3]int{
		{0, 9, 10}, {0, 10, 5}, {0, 11, 22}, {0, 12, 13}, {0, 13, 23}, {0, 14, 5}, {0, 15, 8},
		{1, 11, 18}, {1, 13, 7}, {2, 11, 34}, {2, 12, 13}, {2, 13, 11},
	} {
		f(c[0], c[1], byte(c[2]))
	}
	for x := 3; x < 6; x++ {
		f(x, 11, 18)
		f(x, 13, 7)
	}
	for x := 6; x < 9; x++ {
		f(x, 11, 18)
		f(x, 13, 12)
		f(x, 15, 7)
	}
	for x := 11; x < 19; x++ {
		f(x, 11, 18)
		f(x, 13, 7)
	}
	for _, c := range [][3]int{
		{5, 13, 24}, {5, 14, 4}, {5, 15, 6}, {9, 11, 19}, {9, 12, 5}, {9, 13, 10}, {9, 14, 5},
		{9, 15, 8}, {10, 11, 17}, {10, 12, 4}, {10, 13, 6}, {19, 11, 19}, {19, 12, 5}, {19, 13, 8},
	} {
		f(c[0], c[1], byte(c[2]))
	}
	putGlyphAt(w, 1, 12, 34)
	for i := 0; i < 6; i++ {
		putCharAt(w, i+3, 12, byte(i+24))
	}
	setCursor(w, 1, 14)
	printMessage(3957, w, 18)
	for _, r := range [][3]int{{10, 14, 10}, {1, 15, 4}, {10, 15, 10}} {
		setCursor(w, r[0], r[1])
		for i := 0; i < r[2]; i++ {
			putCharHi(w, ' ', 3)
		}
	}
	resumeWindowDraw(w)
	refreshWindow(w)
}

// wpcvwUnhighlightRealm (WPCVW 7148) redraws realm r's glyph plain (r -1: the exit glyph).
func wpcvwUnhighlightRealm(w, book *Window, r int) {
	if r == -1 {
		putGlyphAt(book, 1, 12, 34)
		return
	}
	putGlyphAt(w, r/3*10+1, r%3<<1+14, byte(r+35))
}

// wpcvwCountKnownSpells (WPCVW 76ee) lists in out the spells of realm r member m knows.
func wpcvwCountKnownSpells(m, r int, out []int) int {
	n := 0
	for s := range spellTable {
		if int(spellTable[s].Realm) == r && testBit(party[m].KnownSpells[:], s) {
			out[n] = s
			n++
		}
	}
	return n
}

// wpcvwDrawSpellList (WPCVW 7749) shows the spells of list around index at, the one at at
// highlighted in realm r's colour when hi.
func wpcvwDrawSpellList(w *Window, r, at int, hi bool, n int, list []int) {
	wpcvwDrawSpellListFrame(w, false)
	deferWindowDraw(w)
	for d := -3; d <= 3; d++ {
		if at+d < 0 || at+d >= n {
			continue
		}
		setCursor(w, 1, d+3)
		c := 18
		if hi && d == 0 {
			c = realmColors[r]
		}
		printMessage(list[at+d]+4000, w, c)
		if hi && d == 0 {
			for w.CurX < 18 {
				putChar(w, ' ', realmColors[r])
			}
		}
	}
	resumeWindowDraw(w)
	refreshWindow(w)
}

// pvSetBlink sets the blinking cursor at (x, y) of w: phase 0 shows a in colour ca,
// phase 1 b in colour cb.
func pvSetBlink(w *Window, x, y int, a string, ca int, b string, cb int) {
	pvBlinkWin, pvBlinkX, pvBlinkY = w, x, y
	pvBlinkText = [2]string{a, b}
	pvBlinkColor = [2]int{ca, cb}
}

// wpcvwChooseSpell (choose_spell, WPCVW 780a) opens member m's spell book: pick a realm,
// then a known spell, then with askPower a power level the member can pay. It returns
// power<<8 + spell, the spell alone without askPower, or -1. A spell cast on a member
// also asks for that member (pvSpellTarget).
func wpcvwChooseSpell(sheet, menu *Window, m int, askPower bool) int {
	p := &party[m]
	book := createWindow(20, 4, 20, 16, 22, -1, 0, false)
	list := createWindow(21, 7, 19, 8, 23, -1, 0, false)
	done := func(v int) int {
		pvBlinkState = -1
		destroyWindow(book, false)
		destroyWindow(list, false)
		return v
	}
	fillWindow(list, ' ', 3, true)
	wpcvwDrawSpellWindow(book)
	wpcvwDrawSpellListFrame(list, true)
	refreshWindow(menu)
	realm := 0
	for r := 0; r < 6; r++ {
		x, y := int(sheet.X)+r/3*10+1, int(sheet.Y)+r%3<<1+14
		setRegion(r, x, y, x, y)
	}
	setRegion(6, int(book.X)+1, int(book.Y)+12, int(book.X)+1, int(book.Y)+12)
	var spells [20]int
	for {
		// Pick a realm.
		endRegions(7)
		n, redraw := 0, true
		for {
			if redraw {
				pvBlinkState = 1
				clearWindowArea(book, 11, 12, 18, 12)
				setCursor(book, 11, 12)
				if realm == -1 {
					pvBlinkWin, pvBlinkX, pvBlinkY = book, 1, 12
					printMessage(3956, book, 9)
					clearWindowArea(book, 6, 14, 8, 14)
					n = -1
					wpcvwDrawSpellListFrame(list, true)
				} else {
					pvBlinkWin, pvBlinkX, pvBlinkY = sheet, realm/3*10+1, realm%3<<1+14
					printMessage(realm+3950, book, realmColors[realm])
					clearWindowArea(book, 6, 14, 8, 14)
					n = wpcvwCountKnownSpells(m, realm, spells[:])
					wpcvwDrawSpellList(list, realm, 0, false, n, spells[:])
				}
				pvSetBlink(pvBlinkWin, pvBlinkX, pvBlinkY, string(rune(realm+35)), 17, "c", 5)
				redraw = false
			}
			k := wpcvwGetMenuInput()
			if menuClicked == 1 {
				switch {
				case k == realm:
					k = 5
				case k < 6:
					wpcvwUnhighlightRealm(sheet, book, realm)
					realm, k = k, 0
				default:
					wpcvwUnhighlightRealm(sheet, book, realm)
					realm, k, n = -1, 5, -1
				}
				redraw = true
			} else {
				switch k {
				case 1:
					if realm == -1 {
						wpcvwUnhighlightRealm(sheet, book, realm)
						realm, redraw = 4, true
					} else if realm-3 >= 0 {
						wpcvwUnhighlightRealm(sheet, book, realm)
						realm, redraw = realm-3, true
					}
				case 2:
					if realm != -1 && realm%3 > 0 {
						wpcvwUnhighlightRealm(sheet, book, realm)
						realm, redraw = realm-1, true
					}
				case 3:
					if realm != -1 {
						wpcvwUnhighlightRealm(sheet, book, realm)
						redraw = true
						if realm += 3; realm >= 6 {
							realm = -1
						}
					}
				case 4:
					if realm != -1 && realm%3+1 < 3 {
						wpcvwUnhighlightRealm(sheet, book, realm)
						realm, redraw = realm+1, true
					}
				}
			}
			if k == 5 && n != 0 {
				break
			}
		}
		if realm == -1 {
			return done(-1)
		}

		// Pick a spell of the realm.
		wpcvwUnhighlightRealm(sheet, book, realm)
		at, back := 0, false
		pvSetBlink(list, 1, 3, wpcvwBlankLabel, 0, "", 0)
		lx, ly := int(list.X), int(list.Y)
		setRegion(7, lx, ly, lx, ly)
		setRegion(8, lx, ly+6, lx, ly+6)
		setRegion(9, lx+1, ly+3, lx+18, ly+3)
		endRegions(10)
		k := 0
		for k != 5 && !back {
			pvBlinkState = 1
			pvBlinkText[1] = getMessage(spells[at] + 4000)
			pvBlinkColor[1] = realmColors[realm]
			wpcvwDrawSpellList(list, realm, at, true, n, spells[:])
			cost := int(spellTable[spells[at]].Cost)
			setCursor(book, 6, 14)
			if int(p.SP[realm].Cur) < cost {
				printString(book, wpcvwNoPoints, realmColors[realm])
			} else {
				printInt(book, cost, 3, realmColors[realm])
			}
			k = wpcvwGetMenuInput()
			if menuClicked == 1 {
				switch {
				case k < 6:
					realm, k = k, 1
				case k == 6:
					return done(-1)
				case k == 7:
					k = 2
				case k == 8:
					k = 4
				case k == 9:
					k = 5
				}
			}
			switch k {
			case 1:
				back = true
			case 2:
				if at > 0 {
					at--
				}
			case 4:
				if at+1 < n {
					at++
				}
			}
		}
		if back {
			continue
		}
		spell := spells[at]
		if !askPower {
			return done(spell)
		}

		// Pick the power level.
		wpcvwDrawSpellListFrame(list, false)
		deferWindowDraw(list)
		setCursor(list, 1, 3)
		printMessage(spell+4000, list, 18)
		resumeWindowDraw(list)
		refreshWindow(list)
		power := 1
		most := min(max(int(p.Level)-int(spellTable[spell].Level)+1, 1), 6)
		bx, by := int(book.X), int(book.Y)
		for i := 0; i < most; i++ {
			setRegion(i, bx+i+3, by+12, bx+i+3, by+12)
		}
		setRegion(most, bx+1, by+12, bx+1, by+12)
		endRegions(most + 1)
		redraw = true
		pvBlinkWin = book
		unmark := func() {
			setCursor(book, power+2, 12)
			putChar(book, byte(power+23), 5)
		}
		for {
			cost := int(spellTable[spell].Cost) * power
			if redraw {
				pvBlinkState, redraw = 1, false
				if power == 0 {
					pvSetBlink(book, 1, 12, "\"", 17, "c", 5)
					clearWindowArea(book, 11, 12, 18, 12)
					setCursor(book, 11, 12)
					printMessage(3956, book, 9)
					clearWindowArea(book, 6, 14, 8, 14)
				} else {
					pvSetBlink(book, power+2, 12, " ", 0, string(rune(power+23)), 7)
					setCursor(book, 6, 14)
					if int(p.SP[realm].Cur) < cost {
						printString(book, wpcvwNoPoints, realmColors[realm])
					} else {
						printInt(book, cost, 3, realmColors[realm])
					}
				}
			}
			k := wpcvwGetMenuInput()
			if menuClicked == 1 {
				if k+1 == power {
					k = 5
				} else {
					unmark()
					power = 0
					if k < most {
						power = k + 1
					}
					k = 0
					if power == 0 {
						k = 5
					}
				}
				redraw = true
			} else {
				switch k {
				case 1:
					if power > 0 {
						unmark()
						power--
						redraw = true
					}
				case 3:
					if power == 0 {
						wpcvwUnhighlightRealm(sheet, book, -1)
						clearWindowArea(book, 11, 12, 18, 12)
						setCursor(book, 11, 12)
						printMessage(realm+3950, book, realmColors[realm])
						power, redraw = 1, true
					} else if power < most {
						unmark()
						power++
						redraw = true
					}
				}
			}
			// The cost is the one before this key, as in the C.
			if k == 5 && int(p.SP[realm].Cur) >= cost {
				break
			}
		}
		done(0)
		if power == 0 {
			return -1
		}
		if spellTable[spell].Target == 1 {
			if pvSpellTarget = chooseMember(menu, 502, -1); pvSpellTarget == -1 {
				return -1
			}
		}
		return power<<8 + spell
	}
}

// learnNewSpell (WPCVW 8353) lets member m learn a spell of book b (0 mage, 1 priest,
// 2 psionic, 3 alchemist) at level up, among those it does not know and its book skill
// reaches.
func learnNewSpell(sheet, menu *Window, m, b int) {
	p := &party[m]
	bs := [4][2]int{{8, 28}, {4, 26}, {1, 25}, {2, 27}}[b] // book bit, skill
	bit, skill := bs[0], bs[1]
	known := p.KnownSpells
	p.KnownSpells = [12]uint8{}
	lv := int(p.Skills[skill])/18 + 1
	if p.Skills[skill] >= 98 {
		lv = 7
	}
	n := 0
	for s := range spellTable {
		if bit&int(spellTable[s].Books) != 0 && int(spellTable[s].Level) <= lv && !testBit(known[:], s) {
			setBit(p.KnownSpells[:], s, true)
			n++
		}
	}
	if n == 0 {
		p.SpellsLearn[b] = 0
		p.KnownSpells = known
		return
	}
	fillWindow(menu, ' ', 3, true)
	refreshWindow(menu)
	wpcvwShowCenteredMsg(menu, replaceCharWithString(getMessage(702), getMessage(int(p.Class)+120), '$'), true)
	fillWindow(menu, ' ', 3, true)
	deferWindowDraw(menu)
	setCursor(menu, 1, 1)
	printMessage(703, menu, 18)
	resumeWindowDraw(menu)
	var saved [6]int16
	for r := range p.SP {
		saved[r], p.SP[r].Cur = p.SP[r].Cur, 999
	}
	s := -1
	for s == -1 {
		s = wpcvwChooseSpell(sheet, menu, m, false)
	}
	pvBlinkState = -1
	p.KnownSpells = known
	for r := range p.SP {
		p.SP[r].Cur = saved[r]
	}
	setBit(p.KnownSpells[:], s, true)
	gain := random(int(p.Skills[skill])/20) + int(spellTable[s].Level)<<1 + 1
	sp := &p.SP[spellTable[s].Realm]
	if int(sp.Max)+gain < int(spellTable[s].Cost) {
		gain = int(spellTable[s].Cost) - int(sp.Max)
	}
	sp.Max += int16(gain)
	sp.Cur += int16(gain)
	p.SpellsLearn[b]--
	drawSpellPoints(sheet, &party[m])
}

// doCastMenu (do_cast_menu, WPCVW 86d1) opens member m's spell book; when canCast it
// casts the chosen spell, paying spell points and stamina.
func doCastMenu(sheet, menu *Window, m int, canCast bool) {
	fillWindow(menu, ' ', 3, true)
	deferWindowDraw(menu)
	setCursor(menu, 8, 1)
	printMessage(701, menu, 18)
	resumeWindowDraw(menu)
	if !canCast {
		wpcvwChooseSpell(sheet, menu, m, false)
		pvBlinkState = -1
		return
	}
	v := wpcvwChooseSpell(sheet, menu, m, true)
	if v == -1 {
		return
	}
	pvBlinkState = -1
	p := &party[m]
	power, s := v/256, v%256
	p.SP[spellTable[s].Realm].Cur -= int16(int(spellTable[s].Cost) * power)
	p.Stamina -= int16(random(power) + int(spellTable[s].Level) + power)
	if p.Stamina < 0 {
		p.Stamina = 0
		p.Status[2] = uint8(random(6) + 6)
	}
	castSpell(m, s, power, pvSpellTarget, sheet, menu)
}
