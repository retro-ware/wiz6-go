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

// The spells castSpell carries out (messages 4000+).
const (
	spellEnergyBlast    = 0
	spellBlindingFlash  = 1
	spellFireball       = 2
	spellFireShield     = 3
	spellFireBomb       = 4
	spellLightning      = 5
	spellPrismicMissile = 6
	spellFirestorm      = 7
	spellNuclearBlast   = 8
	spellChillingTouch  = 9
	spellStamina        = 10
	spellTerror         = 11
	spellWeaken         = 12
	spellSlow           = 13
	spellHaste          = 14
	spellCureParalysis  = 15
	spellIceShield      = 16
	spellIceball        = 17
	spellParalyze       = 18
	spellDeepFreeze     = 19
	spellPoison         = 20
	spellMissileShield  = 21
	spellStinkBomb      = 22
	spellAirPocket      = 23
	spellSilence        = 24
	spellPoisonGas      = 25
	spellCurePoison     = 26
	spellWhirlwind      = 27
	spellPurifyAir      = 28
	spellDeadlyPoison   = 29
	spellLevitate       = 30
	spellToxicVapors    = 31
	spellNoxiousFumes   = 32
	spellAsphyxiation   = 33
	spellDeadlyAir      = 34
	spellAcidSplash     = 35
	spellItchingSkin    = 36
	spellArmorShield    = 37
	spellDirection      = 38
	spellBlades         = 40
	spellArmorplate     = 41
	spellWeb            = 42
	spellAcidBomb       = 43
	spellArmormelt      = 44
	spellCreateLife     = 45
	spellCureStone      = 46
	spellMentalAttack   = 47
	spellSleep          = 48
	spellBless          = 49
	spellCureLesserCnd  = 51
	spellDetectSecret   = 53
	spellIdentify       = 54
	spellHoldMonsters   = 55
	spellSaneMind       = 57
	spellPsionicBlast   = 58
	spellIllusion       = 59
	spellDeath          = 61
	spellMindFlay       = 63
	spellHealWounds     = 64
	spellMakeWounds     = 65
	spellMagicMissile   = 66
	spellDispelUndead   = 67
	spellEnchantedBlade = 68
	spellBlink          = 69
	spellMagicScreen    = 70
	spellConjuration    = 71
	spellAntiMagic      = 72
	spellRemoveCurse    = 73
	spellLifesteal      = 74
	spellAstralGate     = 75
	spellWordOfDeath    = 76
	spellResurrection   = 77
	spellDeathWish      = 78
	spellHolyWater      = 79
	spellHelpFood       = 80
	spellMagicFood      = 81
)

// partyCounters indexes: the party's lasting spell effects, by the spell castSpell sets
// each with.
const (
	effectDirection      = 0
	effectArmorplate     = 1
	effectDetectSecret   = 2
	effectEnchantedBlade = 3
	effectMagicScreen    = 4
	effectLevitate       = 5
)

// castSpell (cast_spell, WPCVW 6b3c) casts spell s at power of caster m on member t.
func castSpell(m, s, power, t int, sheet, menu *Window) {
	c := &partyCounters
	q := &party[t]
	lower := func(v *uint8, by int) { *v = uint8(max(int(int16(int(*v)-by)), 0)) } // C int wraps
	switch s {
	case spellDetectSecret:
		c[effectDetectSecret].Duration += int16(power * 50)
		c[effectDetectSecret].B = int16(clampToward(power*5+70, int(c[effectDetectSecret].B), 0))
	case spellEnchantedBlade:
		c[effectEnchantedBlade].Duration += int16(power * 35)
		c[effectEnchantedBlade].B = int16(clampToward((power-1)/2+1, int(c[effectEnchantedBlade].B), 0))
	case spellArmorplate:
		c[effectArmorplate].Duration += int16(power * 50)
		c[effectArmorplate].B = int16(clampToward((power-1)/2+1, int(c[effectArmorplate].B), 0))
	case spellLevitate:
		c[effectLevitate].Duration += int16(power * 5)
	case spellDirection:
		c[effectDirection].Duration += int16(100 * power)
	case spellMagicScreen:
		c[effectMagicScreen].Duration += int16(power * 50)
		c[effectMagicScreen].B = int16(clampToward(power*5, int(c[effectMagicScreen].B), 0))
	case spellStamina:
		q.Stamina = min(q.Stamina+int16(rollDice(formatDice(power<<1, 8, power))), q.StaminaMax)
		refreshCard(pvCardWin, t)
	case spellCureParalysis:
		lower(&q.Status[statusParalyzed], power*25)
		refreshCard(pvCardWin, t)
	case spellCurePoison:
		if int16(int(q.PoisonStr)-power) <= 0 { // C int wraps
			q.PoisonStr, q.Status[statusPoisoned] = 0, 0
		} else {
			q.PoisonStr -= uint8(power)
		}
		if int16(int(q.Status[statusPoisoned])-power*5) <= 0 { // C int wraps
			q.Status[statusPoisoned], q.PoisonStr = 0, 0
		} else {
			q.Status[statusPoisoned] -= uint8(power * 5)
		}
		refreshCard(pvCardWin, t)
	case spellCureStone:
		if q.Condition == conditionStoned && int(int16(power*3)) > 21-int(q.Stats[statVitality]) { // C int wraps
			q.Stats[statVitality]--
			if random(21) > int(q.Stats[statVitality]) {
				q.Condition, q.HP, q.Stamina = conditionDead, 0, 0
			} else {
				q.Condition = conditionOK
			}
			refreshCard(pvCardWin, t)
		}
	case spellSaneMind:
		lower(&q.Status[statusInsane], power*25)
		refreshCard(pvCardWin, t)
	case spellCureLesserCnd:
		for _, k := range []int{1, statusAsleep, 4, 8, 9} {
			lower(&q.Status[k], power<<1)
		}
		refreshCard(pvCardWin, t)
	case spellHealWounds:
		q.HP = min(q.HP+int16(rollDice(formatDice(power, 7, power))), q.HPMax)
		refreshCard(pvCardWin, t)
	case spellResurrection:
		if q.Condition == conditionDead && int(int16(power*3)) > 18-int(q.Stats[statVitality]) { // C int wraps
			q.Stats[statVitality]--
			q.Condition, q.HP, q.Stamina = conditionOK, 1, q.StaminaMax/10
			q.Unk026++
			refreshCard(pvCardWin, t)
		}
	case spellHelpFood:
		p := &party[m]
		p.Stamina = min(p.Stamina+int16(rollDice(formatDice(power, 8, power))), p.StaminaMax)
		p.HP = min(p.HP+int16(rollDice(formatDice(power, 2, 0))), p.HPMax)
	case spellMagicFood:
		p := &party[m]
		for r := range p.SP {
			p.SP[r].Cur = min(p.SP[r].Cur+int16(rollDice(formatDice(power, 10, 0))), p.SP[r].Max)
		}
	case spellIdentify:
		if party[m].ItemCount > 0 {
			refreshAllWindows()
			if pvChosenItem = chooseItem(sheet, menu, msgCastOnWhichItem, m, false); pvChosenItem != -1 {
				showItemInfo(menu, m, pvChosenItem, power)
			}
		}
	case spellRemoveCurse:
		// As in the C, removing shifts the list, so the next item is skipped.
		for i := 0; i < int(q.ItemCount); i++ {
			if q.Items[i].Flags&itemCursed != 0 {
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
		for s := range spellTable {
			if testBit(p.KnownSpells[:], s) {
				return s
			}
		}
		return -1
	}
	book := createWindow(20, 4, 20, 16, 22, borderNone, 0, false)
	list := createWindow(21, 7, 19, 8, 23, borderNone, 0, false)
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
	for r := range realmCount {
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
					printMessage(msgSpellBookCancel, book, 9)
					clearWindowArea(book, 6, 14, 8, 14)
					n = -1
					drawListBox(list, true)
				} else {
					pvBlinkWin, pvBlinkX, pvBlinkY = sheet, realm/3*10+1, realm%3<<1+14
					printMessage(realm+msgRealmBase, book, realmColors[realm])
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
					k = menuKeyEnter
				case k < realmCount:
					unhighlightSheetRealm(sheet, book, realm)
					realm, k = k, menuKeyEscape
				default:
					unhighlightSheetRealm(sheet, book, realm)
					realm, k, n = -1, menuKeyEnter, -1
				}
				redraw = true
			} else {
				switch k {
				case menuKeyLeft:
					if realm == -1 {
						unhighlightSheetRealm(sheet, book, realm)
						realm, redraw = 4, true
					} else if realm-3 >= 0 {
						unhighlightSheetRealm(sheet, book, realm)
						realm, redraw = realm-3, true
					}
				case menuKeyUp:
					if realm != -1 && realm%3 > 0 {
						unhighlightSheetRealm(sheet, book, realm)
						realm, redraw = realm-1, true
					}
				case menuKeyRight:
					if realm != -1 {
						unhighlightSheetRealm(sheet, book, realm)
						redraw = true
						if realm += 3; realm >= realmCount {
							realm = -1
						}
					}
				case menuKeyDown:
					if realm != -1 && realm%3+1 < 3 {
						unhighlightSheetRealm(sheet, book, realm)
						realm, redraw = realm+1, true
					}
				}
			}
			if k == menuKeyEnter && n != 0 {
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
		for k != menuKeyEnter && !back {
			pvBlinkState = 1
			pvBlinkText[1] = getMessage(spells[at] + msgSpellNameBase)
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
				case k < realmCount:
					realm, k = k, menuKeyLeft
				case k == 6:
					return done(-1)
				case k == 7:
					k = menuKeyUp
				case k == 8:
					k = menuKeyDown
				case k == 9:
					k = menuKeyEnter
				}
			}
			switch k {
			case menuKeyLeft:
				back = true
			case menuKeyUp:
				if at > 0 {
					at--
				}
			case menuKeyDown:
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
		printMessage(spell+msgSpellNameBase, list, 18)
		resumeWindowDraw(list)
		refreshWindow(list)
		power := 1
		most := min(max(int(p.Level)-int(spellTable[spell].Level)+1, 1), maxPower)
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
					printMessage(msgSpellBookCancel, book, 9)
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
					k = menuKeyEnter
				} else {
					unmark()
					power = 0
					if k < most {
						power = k + 1
					}
					k = menuKeyEscape
					if power == 0 {
						k = menuKeyEnter
					}
				}
				redraw = true
			} else {
				switch k {
				case menuKeyLeft:
					if power > 0 {
						unmark()
						power--
						redraw = true
					}
				case menuKeyRight:
					if power == 0 {
						unhighlightSheetRealm(sheet, book, -1)
						clearWindowArea(book, 11, 12, 18, 12)
						setCursor(book, 11, 12)
						printMessage(realm+msgRealmBase, book, realmColors[realm])
						power, redraw = 1, true
					} else if power < most {
						unmark()
						power++
						redraw = true
					}
				}
			}
			// The cost is the one before this key, as in the C.
			if k == menuKeyEnter && int(p.SP[realm].Cur) >= cost {
				break
			}
		}
		done(0)
		if power == 0 {
			return -1
		}
		if spellTable[spell].Target == spellTargetMember {
			if pvSpellTarget = chooseMember(menu, msgSheetCastOnWho, -1); pvSpellTarget == -1 {
				return -1
			}
		}
		return power<<8 + spell
	}
}

// learnNewSpell (WPCVW 8353) lets member m learn a spell of book b (bookMage..bookPsionic)
// at level up, among those it does not know and its book skill reaches.
func learnNewSpell(sheet, menu *Window, m, b int) {
	p := &party[m]
	bit := int(spellBookBits[b])
	skill := [bookCount]int{skillThaumaturgy, skillTheology, skillAlchemy, skillTheosophy}[b]
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
	showCenteredMsg(menu, replaceCharWithString(getMessage(msgYouLearnedNewSpell), getMessage(int(p.Class)+msgClassBase), '$'), true)
	fillWindow(menu, ' ', 3)
	deferWindowDraw(menu)
	setCursor(menu, 1, 1)
	printMessage(msgSelectNewSpellFor, menu, 18)
	resumeWindowDraw(menu)
	var saved [realmCount]int16
	for r := range p.SP {
		saved[r], p.SP[r].Cur = p.SP[r].Cur, unlimitedSP
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
	printMessage(msgSelectSpellCast, menu, 18)
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
		p.Status[statusAsleep] = uint8(random(6) + 6)
	}
	castSpell(m, s, power, pvSpellTarget, sheet, menu)
}
