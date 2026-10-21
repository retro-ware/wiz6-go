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
	return int(int16(v)) // C int wraps
}

// refreshCard (WPCVW 6ac3) shows member m's changed card for a moment.
func refreshCard(w *Window, m int) {
	if dtLog != nil { // stubbed in the difftest
		return
	}
	resumeWindowDraw(w)
	drawMemberCard(w, m)
	waitSecondsOrEnter(4)
	deferWindowDraw(w)
}

// castSpell (cast_spell, WPCVW 6b3c) casts spell s at power of caster m on member t.
func castSpell(m, s, power, t int, sheet, menu *Window) {
	c := &partyCounters
	q := &party[t]
	lower := func(v *uint8, by int) { *v = uint8(max(int(int16(int(*v)-by)), 0)) } // C int wraps
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
		refreshCard(pvCardWin, t)
	case 15:
		lower(&q.Status[3], power*25)
		refreshCard(pvCardWin, t)
	case 26:
		if int16(int(q.PoisonStr)-power) <= 0 { // C int wraps
			q.PoisonStr, q.Status[5] = 0, 0
		} else {
			q.PoisonStr -= uint8(power)
		}
		if int16(int(q.Status[5])-power*5) <= 0 { // C int wraps
			q.Status[5], q.PoisonStr = 0, 0
		} else {
			q.Status[5] -= uint8(power * 5)
		}
		refreshCard(pvCardWin, t)
	case 46:
		if q.Condition == 2 && int(int16(power*3)) > 21-int(q.Stats[3]) { // C int wraps
			q.Stats[3]--
			if random(21) > int(q.Stats[3]) {
				q.Condition, q.HP, q.Stamina = 3, 0, 0
			} else {
				q.Condition = 0
			}
			refreshCard(pvCardWin, t)
		}
	case 57:
		lower(&q.Status[7], power*25)
		refreshCard(pvCardWin, t)
	case 51:
		for _, k := range []int{1, 2, 4, 8, 9} {
			lower(&q.Status[k], power<<1)
		}
		refreshCard(pvCardWin, t)
	case 64:
		q.HP = min(q.HP+int16(rollDice(formatDice(power, 7, power))), q.HPMax)
		refreshCard(pvCardWin, t)
	case 77:
		if q.Condition == 3 && int(int16(power*3)) > 18-int(q.Stats[3]) { // C int wraps
			q.Stats[3]--
			q.Condition, q.HP, q.Stamina = 0, 1, q.StaminaMax/10
			q.Unk026++
			refreshCard(pvCardWin, t)
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
				removeItemShown(t, i)
			}
		}
	}
}

// countKnownSpells (WPCVW 76ee) lists in out the spells of realm r member m knows.
func countKnownSpells(m, r int, out []int) int {
	n := 0
	for s := range spellTable {
		if int(spellTable[s].Realm) == r && testBit(party[m].KnownSpells[:], s) {
			out[n] = s
			n++
		}
	}
	return n
}

// pvSetBlink sets the blinking cursor at (x, y) of w: phase 0 shows a in colour ca,
// phase 1 b in colour cb.
func pvSetBlink(w *Window, x, y int, a string, ca int, b string, cb int) {
	pvBlinkWin, pvBlinkX, pvBlinkY = w, x, y
	pvBlinkText = [2]string{a, b}
	pvBlinkColor = [2]int{ca, cb}
}

// chooseSheetSpell (choose_spell, WPCVW 780a) opens member m's spell book: pick a realm,
// then a known spell, then with askPower a power level the member can pay. It returns
// power<<8 + spell, the spell alone without askPower, or -1. A spell cast on a member
// also asks for that member (pvSpellTarget).
func chooseSheetSpell(sheet, menu *Window, m int, askPower bool) int {
	p := &party[m]
	if dtLog != nil { // stubbed in the difftest: the lowest known spell
		for s := range 82 {
			if testBit(p.KnownSpells[:], s) {
				return s
			}
		}
		return -1
	}
	book := createWindow(20, 4, 20, 16, 22, -1, 0, false)
	list := createWindow(21, 7, 19, 8, 23, -1, 0, false)
	done := func(v int) int {
		pvBlinkState = -1
		destroyWindow(book, false)
		destroyWindow(list, false)
		return v
	}
	fillWindow(list, ' ', 3)
	drawSheetSpellWindow(book)
	drawListBox(list, true)
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
					drawListBox(list, true)
				} else {
					pvBlinkWin, pvBlinkX, pvBlinkY = sheet, realm/3*10+1, realm%3<<1+14
					printMessage(realm+3950, book, realmColors[realm])
					clearWindowArea(book, 6, 14, 8, 14)
					n = countKnownSpells(m, realm, spells[:])
					drawSheetSpellList(list, realm, 0, false, spells[:n])
				}
				pvSetBlink(pvBlinkWin, pvBlinkX, pvBlinkY, string(rune(realm+35)), 17, "c", 5)
				redraw = false
			}
			k := getMenuInputBlink()
			if menuClicked == 1 {
				switch {
				case k == realm:
					k = 5
				case k < 6:
					unhighlightSheetRealm(sheet, book, realm)
					realm, k = k, 0
				default:
					unhighlightSheetRealm(sheet, book, realm)
					realm, k, n = -1, 5, -1
				}
				redraw = true
			} else {
				switch k {
				case 1:
					if realm == -1 {
						unhighlightSheetRealm(sheet, book, realm)
						realm, redraw = 4, true
					} else if realm-3 >= 0 {
						unhighlightSheetRealm(sheet, book, realm)
						realm, redraw = realm-3, true
					}
				case 2:
					if realm != -1 && realm%3 > 0 {
						unhighlightSheetRealm(sheet, book, realm)
						realm, redraw = realm-1, true
					}
				case 3:
					if realm != -1 {
						unhighlightSheetRealm(sheet, book, realm)
						redraw = true
						if realm += 3; realm >= 6 {
							realm = -1
						}
					}
				case 4:
					if realm != -1 && realm%3+1 < 3 {
						unhighlightSheetRealm(sheet, book, realm)
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
		unhighlightSheetRealm(sheet, book, realm)
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
			drawSheetSpellList(list, realm, at, true, spells[:n])
			cost := int(spellTable[spells[at]].Cost)
			setCursor(book, 6, 14)
			if int(p.SP[realm].Cur) < cost {
				printString(book, wpcvwNoPoints, realmColors[realm])
			} else {
				printInt(book, cost, 3, realmColors[realm])
			}
			k = getMenuInputBlink()
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
		drawListBox(list, false)
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
			k := getMenuInputBlink()
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
						unhighlightSheetRealm(sheet, book, -1)
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
	fillWindow(menu, ' ', 3)
	refreshWindow(menu)
	showCenteredMsg(menu, replaceCharWithString(getMessage(702), getMessage(int(p.Class)+120), '$'), true)
	fillWindow(menu, ' ', 3)
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
		s = chooseSheetSpell(sheet, menu, m, false)
	}
	pvBlinkState = -1
	p.KnownSpells = known
	for r := range p.SP {
		p.SP[r].Cur = saved[r]
	}
	setBit(p.KnownSpells[:], s, true)
	gain := random(int(p.Skills[skill])/20) + int(spellTable[s].Level)<<1 + 1
	sp := &p.SP[spellTable[s].Realm]
	if int(int16(int(sp.Max)+gain)) < int(spellTable[s].Cost) { // C int sum wraps
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
	fillWindow(menu, ' ', 3)
	deferWindowDraw(menu)
	setCursor(menu, 8, 1)
	printMessage(701, menu, 18)
	resumeWindowDraw(menu)
	if !canCast {
		chooseSheetSpell(sheet, menu, m, false)
		pvBlinkState = -1
		return
	}
	v := chooseSheetSpell(sheet, menu, m, true)
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
