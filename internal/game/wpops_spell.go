package game

// WPOPS: the spell book and the item list of a combat action. WDOPT has the same
// spell-window functions and uses these.

// ROOT data g02ca: the spell book frame, 49 entries of column, row, glyph and an end
// (column or, with bit 7, row) the glyph repeats up to; end 0 is a single glyph.
var spellFrame = [49][4]uint8{
	{0, 0, 1, 0}, {21, 0, 3, 0}, {0, 6, 6, 0}, {21, 6, 8, 0}, {1, 0, 2, 12}, {1, 6, 7, 12},
	{13, 0, 2, 21}, {15, 2, 12, 21}, {18, 4, 12, 21}, {18, 6, 7, 21}, {0, 1, 4, 134},
	{21, 1, 5, 134}, {1, 2, 12, 12}, {1, 4, 12, 12}, {2, 1, 13, 134}, {6, 1, 13, 134},
	{8, 1, 13, 134}, {12, 1, 13, 132}, {12, 4, 5, 134}, {17, 5, 4, 0}, {17, 6, 6, 0},
	{0, 2, 9, 0}, {0, 4, 9, 0}, {21, 2, 10, 0}, {21, 4, 10, 0}, {2, 0, 25, 0}, {6, 0, 25, 0},
	{8, 0, 25, 0}, {12, 0, 25, 0}, {2, 6, 11, 0}, {6, 6, 11, 0}, {8, 6, 11, 0}, {12, 6, 8, 0},
	{2, 2, 16, 0}, {2, 4, 16, 0}, {6, 2, 16, 0}, {6, 4, 16, 0}, {8, 2, 16, 0}, {8, 4, 16, 0},
	{12, 2, 16, 0}, {12, 4, 23, 0}, {13, 2, 12, 0}, {13, 4, 7, 0}, {14, 2, 14, 0},
	{14, 3, 13, 0}, {14, 4, 11, 0}, {15, 4, 7, 0}, {16, 4, 7, 0}, {17, 4, 24, 0},
}

// WPOPS data 0x4fee and 0x4ffe: the blank spell-name blink and the "not enough points" mark.
const (
	spellBlank  = "               "
	spellNoCost = "***"
)

// unhighlightBookRealm (unhighlight_combat_realm, WPOPS 5fd6; WDOPT 6789, WTREA 74f6)
// redraws realm r's icon (-1 the cancel icon) unlit.
func unhighlightBookRealm(w *Window, r int) {
	if r == -1 {
		putGlyphAt(w, 13, 3, 34)
		return
	}
	putGlyphAt(w, r/3*6+1, r%3<<1+1, byte(r+35))
}

// drawBookSpellFrame (draw_spell_frame, WPOPS 60cd; WDOPT 6880, WTREA 75ed).
func drawBookSpellFrame(w *Window) {
	for _, e := range spellFrame {
		end, vertical := int(e[3]%128), e[3]/128 != 0
		switch {
		case end == 0:
			setFrameGlyph(w, int(e[0]), int(e[1]), e[2])
		case !vertical:
			for x := int(e[0]); x < end; x++ {
				setFrameGlyph(w, x, int(e[1]), e[2])
			}
		default:
			for y := int(e[1]); y < end; y++ {
				setFrameGlyph(w, int(e[0]), y, e[2])
			}
		}
	}
}

// drawBookSpellWindow (draw_combat_spell_window, WPOPS 61a0; WDOPT 6953, WTREA 76c0)
// draws the spell book: the realms with member m's spell points, the power levels and
// the cancel icon.
func drawBookSpellWindow(w *Window, m int) {
	deferWindowDraw(w)
	fillWindow(w, 0, 1)
	drawBookSpellFrame(w)
	for _, g := range [][3]int{{1, 1, 35}, {1, 3, 36}, {1, 5, 37}, {7, 1, 38}, {7, 3, 39}, {7, 5, 40}, {13, 3, 34}} {
		putGlyphAt(w, g[0], g[1], byte(g[2]))
	}
	for r := range 6 {
		x := 3
		if r >= 3 {
			x = 9
		}
		setCursor(w, x, r%3<<1+1)
		printInt(w, int(party[m].SP[r].Cur), 3, realmColors[r])
	}
	for i := range 6 {
		putCharAt(w, i+15, 3, byte(i+24))
	}
	setCursor(w, 13, 5)
	printMessage(msgCost, w, 18)
	setCursor(w, 13, 6)
	for range 4 {
		putCharHi(w, ' ', 3)
	}
	resumeWindowDraw(w)
	refreshWindow(w)
}

// drawBookSpellList (draw_combat_spell_list, WPOPS 638a; WDOPT 6b3d, WTREA 78aa) shows
// the spells of list (n of them, of realm) around pos, pos lit with hi.
func drawBookSpellList(w *Window, realm, pos int, hi bool, n int, list []int) {
	drawListFrame(w, false)
	deferWindowDraw(w)
	for r := -2; r <= 2; r++ {
		i := pos + r
		if i < 0 || i >= n {
			continue
		}
		setCursor(w, 2, r+2)
		lit := hi && r == 0
		color := 18
		if lit {
			color = realmColors[realm]
		}
		printMessage(list[i]+msgSpellNameBase, w, color)
		for lit && w.CurX < 17 {
			putChar(w, ' ', realmColors[realm])
		}
	}
	resumeWindowDraw(w)
	refreshWindow(w)
}

// chooseCombatSpell (WPOPS 644a) picks a realm, a spell and its power for member m and
// the spell's target; (power<<8)+spell, or -1.
func chooseCombatSpell(win *Window, m int) int {
	book := createWindow(0, 18, 23, 7, 25, borderNone, 0, false)
	list := createWindow(22, 19, 18, 6, 26, borderTopEdge, 0, false)
	drawBookSpellWindow(book, m)
	drawListFrame(list, true)
	region := func(i, x, y int) {
		setClickRegion(i, book.X+byte(x), book.Y+byte(y), book.X+byte(x), book.Y+byte(y))
	}
	for i := range 6 {
		region(i, i/3*6+1, i%3<<1+1)
	}
	region(6, 13, 3)
	quit := func() int {
		pvBlinkState = -1
		destroyWindow(book, false)
		destroyWindow(list, false)
		return -1
	}
	var spells [82]int
	realm, count, pos := 0, 0, 0
	sp := func() int { return int(party[m].SP[realm].Cur) }
	for chosen := false; !chosen; {
		// The realm.
		setClickRegion(7, clickRegionsEnd, 0, 0, 0)
		redraw := true
		pvBlinkWin = book
		for {
			if redraw {
				pvBlinkState = 1
				clearWindowArea(book, 13, 1, 20, 1)
				setCursor(book, 13, 1)
				if realm == -1 {
					pvBlinkX, pvBlinkY = 13, 3
					printMessage(msgSpellBookCancel, book, 9)
					clearWindowArea(book, 18, 5, 20, 5)
					count = -1
					drawListFrame(list, true)
				} else {
					pvBlinkX, pvBlinkY = realm/3*6+1, realm%3<<1+1
					printMessage(realm+msgRealmBase, book, realmColors[realm])
					clearWindowArea(book, 18, 5, 20, 5)
					count = countKnownSpells(m, realm, spells[:])
					drawBookSpellList(list, realm, 0, false, count, spells[:])
				}
				pvBlinkText = [2]string{string(rune(realm + 35)), "c"}
				pvBlinkColor = [2]int{17, 5}
				redraw = false
			}
			k := getCombatInput()
			if menuClicked == 1 {
				if k == realm {
					k = 5
				} else {
					unhighlightBookRealm(book, realm)
					if k < 6 {
						realm, k = k, 0
					} else {
						realm, k, count = -1, 5, -1
					}
				}
				redraw = true
			} else {
				switch k {
				case 1:
					if realm == -1 {
						unhighlightBookRealm(book, realm)
						realm, redraw = 4, true
					} else if realm-3 >= 0 {
						unhighlightBookRealm(book, realm)
						realm, redraw = realm-3, true
					}
				case 2:
					if realm != -1 && realm%3 > 0 {
						unhighlightBookRealm(book, realm)
						realm, redraw = realm-1, true
					}
				case 3:
					if realm != -1 {
						unhighlightBookRealm(book, realm)
						redraw = true
						if realm+3 >= 6 {
							realm = -1
						} else {
							realm += 3
						}
					}
				case 4:
					if realm != -1 && realm%3+1 < 3 {
						unhighlightBookRealm(book, realm)
						realm, redraw = realm+1, true
					}
				}
			}
			if k == 5 && count != 0 {
				break
			}
		}
		if realm == -1 {
			return quit()
		}
		// The spell.
		unhighlightBookRealm(book, realm)
		pos = 0
		pvBlinkWin, pvBlinkX, pvBlinkY = list, 2, 2
		pvBlinkText[0], pvBlinkColor[0] = spellBlank, 0
		setClickRegion(7, list.X+1, list.Y, list.X+1, list.Y)
		setClickRegion(8, list.X+1, list.Y+4, list.X+1, list.Y+4)
		setClickRegion(9, list.X+2, list.Y+2, list.X+19, list.Y+2)
		setClickRegion(10, clickRegionsEnd, 0, 0, 0)
		chosen = true
		for k := 0; k != 5; {
			pvBlinkState = 1
			pvBlinkText[1] = getMessage(spells[pos] + msgSpellNameBase)
			pvBlinkColor[1] = realmColors[realm]
			drawBookSpellList(list, realm, pos, true, count, spells[:])
			cost := int(spellTable[spells[pos]].Cost)
			setCursor(book, 18, 5)
			if sp() < cost {
				printString(book, spellNoCost, realmColors[realm])
			} else {
				printInt(book, cost, 3, realmColors[realm])
			}
			k = getCombatInput()
			if menuClicked == 1 {
				switch k {
				case 0, 1, 2, 3, 4, 5:
					realm, k = k, 1
				case 6:
					return quit()
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
				chosen = false
			case 2:
				if pos > 0 {
					pos--
				}
			case 4:
				if pos+1 < count {
					pos++
				}
			}
			if !chosen {
				break
			}
		}
	}
	// The power level.
	spell := spells[pos]
	drawListFrame(list, false)
	deferWindowDraw(list)
	setCursor(list, 2, 2)
	printMessage(spell+msgSpellNameBase, list, 18)
	resumeWindowDraw(list)
	refreshWindow(list)
	pvBlinkWin = book
	level := 1
	maxLevel := min(max(int(party[m].Level)-int(spellTable[spell].Level)+1, 1), 6)
	for i := range maxLevel {
		region(i, i+15, 3)
	}
	region(maxLevel, 13, 3)
	setClickRegion(maxLevel+1, clickRegionsEnd, 0, 0, 0)
	unlight := func() {
		setCursor(book, level+14, 3)
		putChar(book, byte(level+23), 5)
	}
	rc := realmColors[realm]
	for redraw := true; ; {
		cost := int(spellTable[spell].Cost) * level
		if redraw {
			redraw = false
			pvBlinkState = 1
			if level == 0 {
				pvBlinkX, pvBlinkY = 13, 3
				pvBlinkText = [2]string{"\x22", "c"}
				pvBlinkColor = [2]int{17, 5}
				clearWindowArea(book, 13, 1, 20, 1)
				setCursor(book, 13, 1)
				printMessage(msgSpellBookCancel, book, 9)
				clearWindowArea(book, 18, 5, 20, 5)
			} else {
				pvBlinkX, pvBlinkY = level+14, 3
				pvBlinkText = [2]string{" ", string(rune(level + 23))}
				pvBlinkColor = [2]int{0, 7}
				setCursor(book, 18, 5)
				if sp() < cost {
					printString(book, spellNoCost, rc)
				} else {
					printInt(book, cost, 3, rc)
				}
			}
		}
		k := getCombatInput()
		if menuClicked == 1 {
			if k+1 == level {
				k = 5
			} else {
				unlight()
				if k < maxLevel {
					level, k = k+1, 0
				} else {
					level, k = 0, 5
				}
			}
			redraw = true
		} else {
			switch k {
			case 1:
				if level > 0 {
					unlight()
					level--
					redraw = true
				}
			case 3:
				if level == 0 {
					unhighlightBookRealm(book, -1)
					clearWindowArea(book, 13, 1, 20, 1)
					setCursor(book, 13, 1)
					printMessage(realm+msgRealmBase, book, rc)
					level, redraw = 1, true
				} else if level < maxLevel {
					unlight()
					level++
					redraw = true
				}
			}
		}
		if k == 5 && sp() >= cost {
			break
		}
	}
	quit()
	if level == 0 {
		return -1
	}
	c := &monsterGroups[0].Members[m]
	switch spellTable[spell].Target {
	case spellTargetMember:
		fillWindow(win, 32, 3)
		refreshAllWindows()
		c.TargetGroup = 0
		t := selectPartyMember(msgCombatCastOnWho)
		if t == -1 {
			return -1
		}
		c.TargetMember = int8(t)
	case spellTargetGroup:
		t := selectMonsterGroup(win, msgCastOn, 3, 0)
		if t == -1 {
			return -1
		}
		c.TargetGroup = int8(t)
	}
	return level<<8 + spell
}

// drawItemList (draw_item_list, WPOPS 7017) shows member m's items around pos,
// pos lit with hi, with the quantity of stackable ones and the item icon.
func drawItemList(w *Window, m, pos int, hi bool, names []string, counts []int) {
	drawListFrame(w, false)
	deferWindowDraw(w)
	for r := -2; r <= 2; r++ {
		i := pos + r
		if i < 0 || int(party[m].ItemCount) <= i {
			continue
		}
		setCursor(w, 2, r+2)
		if hi && r == 0 {
			printString(w, names[i], 3)
			if counts[i] > 0 {
				for w.CurX < 15 {
					putChar(w, ' ', 3)
				}
				printInt(w, counts[i], 3, 1)
			} else {
				for w.CurX < 18 {
					putChar(w, ' ', 3)
				}
			}
		} else {
			printStringHi(w, names[i], 3)
			if counts[i] > 0 {
				setCursor(w, 15, r+2)
				printInt(w, counts[i], 3, 18)
			}
		}
		setCursor(w, 18, r+2)
		putCharHi(w, party[m].Items[i].Icon+1, 4)
	}
	resumeWindowDraw(w)
	refreshWindow(w)
}

// chooseCombatItem (WPOPS 71be) picks one of member m's items to use, and the target of
// its spell; the item, or -1.
func chooseCombatItem(win *Window, m int) int {
	prompt := createWindow(0, 19, 19, 5, 12, borderRightEdge, 0, false)
	fillWindow(prompt, 32, 3)
	list := createWindow(20, 19, 20, 5, 20, borderTopEdge, 0, false)
	fillWindow(list, 32, 3)
	deferWindowDraw(prompt)
	setCursor(prompt, 2, 1)
	printMessage(msgCombatUseWhichItem, prompt, 18)
	resumeWindowDraw(prompt)
	drawListFrame(list, false)
	refreshAllWindows()
	p := &party[m]
	names, counts := make([]string, p.ItemCount), make([]int, p.ItemCount)
	for i := range names {
		rec := itemRec(int(p.Items[i].ID))
		names[i] = cstr(rec.Name[:])
		if stackable[p.Items[i].Type] == 1 {
			counts[i] = int(p.Items[i].Qty)
		}
	}
	cancel := getMessage(msgCombatUseCancel)
	pos, onCancel := 0, false
	setClickRegion(0, list.X+2, list.Y+2, list.X+19, list.Y+2)
	setClickRegion(1, list.X+1, list.Y, list.X+1, list.Y)
	setClickRegion(2, list.X+1, list.Y+4, list.X+1, list.Y+4)
	setClickRegion(3, prompt.X+7, prompt.Y+3, byte(len(cancel)+int(prompt.X)+6), prompt.Y+3)
	setClickRegion(4, clickRegionsEnd, 0, 0, 0)
	for k := 0; k != 5; {
		setCursor(prompt, 7, 3)
		if onCancel {
			printString(prompt, cancel, -3)
		} else {
			printStringHi(prompt, cancel, 3)
		}
		drawItemList(list, m, pos, !onCancel, names, counts)
		k = getCombatInput()
		if menuClicked == 1 {
			switch k {
			case 0:
				k = 5
			case 1:
				k = 2
			case 2:
				k = 4
			case 3:
				onCancel, k = true, 5
			}
		}
		switch k {
		case 1:
			onCancel = true
		case 2:
			if onCancel {
				onCancel = false
			} else if pos > 0 {
				pos--
			}
		case 3:
			onCancel = false
		case 4:
			if onCancel {
				onCancel = false
			} else if int(p.ItemCount) > pos+1 {
				pos++
			}
		}
	}
	destroyWindow(prompt, false)
	destroyWindow(list, false)
	if onCancel {
		return -1
	}
	rec := itemRec(int(p.Items[pos].ID))
	if rec.Spell != 0 {
		spell := int(rec.Spell) % 256
		c := &monsterGroups[0].Members[m]
		switch spellTable[spell].Target {
		case spellTargetMember:
			fillWindow(win, 32, 3)
			refreshAllWindows()
			c.TargetGroup = 0
			t := selectPartyMember(msgCombatUseOnWho)
			if t == -1 {
				return -1
			}
			c.TargetMember = int8(t)
		case spellTargetGroup:
			t := selectMonsterGroup(win, msgUseOn, 3, 0)
			if t == -1 {
				return -1
			}
			c.TargetGroup = int8(t)
		}
	}
	return pos
}
