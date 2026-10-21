package game

// WPCVW: the character sheet, its panels and its main menu.

// drawMemberCard (WPCVW 49c9) draws member m's card in w: name, portrait, state, class,
// hands and the HP and stamina gauges.
func drawMemberCard(w *Window, m int) {
	p := &party[m]
	x, y := m%2*9+2, m/2*4+2
	setCursor(w, x, y)
	printStringHi(w, memberName(m), 3)
	for i := len(memberName(m)); i < 7; i++ {
		putCharHi(w, ' ', 3)
	}
	drawPortrait(w, int(memberPortrait[m]), x, y+1)
	g := conditionGlyph[p.Condition]
	if g == 0 && p.Status[2] > 0 {
		g = 1
	}
	if g < 2 && p.Status[3] > 0 {
		g = 2
	}
	setCursor(w, x+3, y+3)
	putCharHi(w, byte(g+37), 1)
	g = -1
	for i := 0; i < 10; i++ {
		if p.Status[i] > 0 && statusGlyph[i] > g {
			g = statusGlyph[i]
		}
	}
	if g != -1 {
		putCharHi(w, byte(g+37), 1)
	} else {
		putCharHi(w, ' ', 3)
	}
	setCursor(w, x+3, y+2)
	for i := 0; i < 2; i++ {
		putCharHi(w, byte(int(p.Class)<<1+i+58), 1)
	}
	setCursor(w, x+3, y+1)
	for i := 0; i < 2; i++ {
		if e := p.Equipped[i]; e != 255 && p.Items[e].ID > 0 {
			putCharHi(w, p.Items[e].Icon+1, 4)
		} else {
			putCharHi(w, byte(i+37), 4)
		}
	}
	drawGauge(w, x+5, y+1, int(p.HP), int(p.HPMax), 86)
	drawGauge(w, x+6, y+1, int(p.Stamina), int(p.StaminaMax), 99)
}

// refreshMemberCard (WPCVW 6ac3) shows member m's card for a moment.
func refreshMemberCard(w *Window, m int) {
	resumeWindowDraw(w)
	drawMemberCard(w, m)
	waitSecondsOrEnter(4)
	deferWindowDraw(w)
}

// drawSheetFrame (WPCVW 4ca8) draws the sheet's frame into its cells.
func drawSheetFrame(w *Window) {
	f := func(x, y int, ch byte) { wpcvwPutFrameChar(w, x, y, ch) }
	f(0, 0, 1)
	f(39, 0, 3)
	f(0, 19, 6)
	f(39, 19, 8)
	for i := 1; i < 39; i++ {
		f(i, 0, 2)
		f(i, 19, 7)
	}
	for i := 1; i < 19; i++ {
		f(0, i, 4)
		f(39, i, 5)
	}
	f(0, 4, 9)
	f(0, 13, 9)
	f(0, 15, 9)
	f(0, 17, 9)
	f(39, 4, 10)
	f(2, 19, 11)
	f(10, 19, 11)
	f(12, 19, 11)
	f(20, 19, 11)
	f(39, 8, 10)
	f(37, 19, 11)
	for i := 1; i < 39; i++ {
		f(i, 4, 12)
	}
	for i := 21; i < 39; i++ {
		f(i, 8, 12)
	}
	f(20, 4, 14)
	for i := 5; i < 19; i++ {
		f(20, i, 13)
	}
	for i := 8; i < 19; i++ {
		f(37, i, 13)
	}
	f(20, 8, 33)
	f(20, 13, 15)
	f(20, 15, 15)
	f(20, 17, 15)
	f(37, 8, 14)
	for i := 1; i < 20; i++ {
		f(i, 13, 12)
		f(i, 15, 12)
		f(i, 17, 12)
	}
	f(2, 13, 14)
	f(10, 13, 14)
	f(12, 13, 14)
	for i := 14; i < 19; i++ {
		f(2, i, 13)
		f(10, i, 13)
		f(12, i, 13)
	}
	for _, x := range []int{2, 10, 12} {
		f(x, 15, 16)
		f(x, 17, 16)
	}
}

// LoadCategory is get_load_category's result: what carrying the current load costs.
type LoadCategory struct {
	AC           int // w[0]: armour class penalty
	A, B         int // w[1], w[2]
	CapA, CapB   int // w[3], w[4]: caps on PC.Unk1A9, Unk1AA
	StaminaDrain int // w[5]
	Color        int // w[6]: the load's colour on the sheet
}

// getLoadCategory (WPCVW 518e) rates member m's load against his capacity.
func getLoadCategory(m int) LoadCategory {
	p := &party[m]
	l := LoadCategory{CapA: int(p.Unk1A9), CapB: int(p.Unk1AA), Color: 9}
	pct := int32(p.Carried) * 100 / int32(p.MaxLoad)
	switch {
	case pct <= 50:
	case pct <= 66:
		l.A, l.B = 5, 10
		l.CapA, l.CapB = min(l.CapA, 3), min(l.CapB, 3)
	case pct <= 75:
		l.AC, l.A, l.B = 1, 10, 20
		l.CapA, l.CapB = min(l.CapA, 2), min(l.CapB, 2)
		l.StaminaDrain, l.Color = int(p.StaminaMax)/100, -7
	case pct <= 90:
		l.AC, l.A, l.B, l.CapA, l.CapB = 2, 20, 40, 1, 1
		l.StaminaDrain, l.Color = int(p.StaminaMax)/50, -5
	default:
		l.AC, l.A, l.B, l.CapA, l.CapB = 4, 40, 80, 1, 1
		l.StaminaDrain, l.Color = int(p.StaminaMax)/25, -4
	}
	return l
}

// getItemWeight (WPCVW 5311) is the weight of item slot i of member m, times its
// quantity for stackable items.
func getItemWeight(i, m int) int16 {
	it := party[m].Items[i]
	qty := int16(1)
	if stackable[it.Type] == 1 {
		if qty = int16(it.Qty); qty < 1 {
			qty = 1
		}
	}
	return it.Weight * qty
}

// calcCarriedWeight (WPCVW 5370) recomputes member m's carried weight.
func calcCarriedWeight(m int) {
	p := &party[m]
	p.Carried = 0
	for i := 0; i < int(p.ItemCount); i++ {
		p.Carried += getItemWeight(i, m)
	}
	for i := 0; i < int(p.SwagCount); i++ {
		p.Carried += getItemWeight(i+10, m)
	}
}

// draw_spell_points (WPCMK 50d1, WPCVW 53fa) prints pc's six realms' spell points on
// the sheet w.
func drawSpellPoints(w *Window, pc *PC) {
	for r := 0; r < 6; r++ {
		x := 13
		if r < 3 {
			x = 3
		}
		setCursor(w, x, r%3<<1+14)
		printInt(w, int(pc.SP[r].Cur), 3, realmColors[r])
		putChar(w, '/', 9)
		printInt(w, int(pc.SP[r].Max), 3, realmColors[r])
	}
}

// drawStatsPanel (WPCVW 54ab) draws the left and top of the sheet: portrait, name, race,
// class, age, level, rank, experience, kills, stats, HP, stamina, state, gold, load and
// spell points.
func drawStatsPanel(w *Window, m int) {
	p := &party[m]
	drawPortrait(w, int(memberPortrait[m]), 1, 1)
	wpcvwPrintAt(w, 4, 1, memberName(m), 5)
	setCursor(w, 13, 1)
	putChar(w, (getMessage(int(p.Sex)+140) + "\x00")[0], 1)
	putChar(w, '-', 1)
	wpcvwPrintMsg(int(p.Race)+100, w, 1)
	setCursor(w, 4, 2)
	putChar(w, 30, 5)
	printInt(w, int(p.AgeDays/365), 3, 14)
	w.CurX++
	if testBit(p.Flags[:], 0) || testBit(p.Flags[:], 1) || testBit(p.Flags[:], 2) {
		putChar(w, 10, 5)
	}
	printMsgAt(int(p.Class)+120, w, 13, 2, 1)
	setCursor(w, 4, 3)
	putChar(w, 31, 5)
	printInt(w, int(p.Unk026), 3, 12)
	printMsgAt(200, w, 13, 3, 8)
	w.CurX++
	printInt(w, int(p.Level), 3, 9)
	printMsgAt(201, w, 25, 1, 11)
	rank := getMessage(int(p.Class)*10 + int(p.Rank) + 800)
	setCursor(w, 39-len(rank), 1)
	printString(w, rank, 3)
	printMsgAt(202, w, 25, 2, 14)
	w.CurX++
	printLong(w, p.Exp, 10, 6)
	printMsgAt(203, w, 25, 3, 14)
	w.CurX++
	printLong(w, p.Kills, 10, 6)
	for i := 0; i < 8; i++ {
		printMsgAt(i+204, w, 1, i+5, 5)
		w.CurX++
		printInt(w, int(p.Stats[i]), 2, 1)
	}
	printMsgAt(212, w, 10, 5, 4)
	w.CurX++
	printInt(w, int(p.HP), 4, 6)
	putChar(w, '/', 8)
	setCursor(w, 14, 6)
	printInt(w, int(p.HPMax), 4, 6)
	printMsgAt(213, w, 10, 8, 4)
	w.CurX += 2
	printInt(w, staminaPct(p), 3, 6)
	putChar(w, '%', 4)
	printMsgAt(214, w, 10, 9, 4)
	w.CurX++
	x := w.CurX
	g := conditionGlyph[p.Condition]
	if g == 0 && p.Status[2] > 0 {
		g = 1
	}
	if g < 2 && p.Status[3] > 0 {
		g = 2
	}
	if g != 0 {
		putCharHi(w, byte(g+47), 1)
	}
	if p.Status[9] > 0 {
		putCharHi(w, byte(statusGlyph[9]+47), 1)
	}
	if p.Status[4] > 0 {
		putCharHi(w, byte(statusGlyph[4]+47), 1)
	}
	if p.Status[5] > 0 || p.Status[6] > 0 || p.Status[8] > 0 {
		putCharHi(w, byte(statusGlyph[5]+47), 1)
	}
	if w.CurX == x {
		putCharHi(w, byte(conditionGlyph[0]+47), 1)
	}
	printMsgAt(215, w, 10, 11, 13)
	w.CurX++
	printLong(w, p.Gold, 7, 5)
	calcCarriedWeight(m)
	load := getLoadCategory(m)
	p.LoadCat = uint8(load.StaminaDrain)
	c := load.Color
	c8 := c
	if c == 9 {
		c8 = 8
	}
	printMsgAt(216, w, 10, 12, c8)
	printInt(w, int(p.Carried)/10, 3, c)
	putChar(w, '/', c8)
	printInt(w, int(p.MaxLoad)/10, 3, c)
	for r := 0; r < 6; r++ {
		putGlyphAt(w, r/3*10+1, r%3*2+14, byte(r+35))
	}
	drawSpellPoints(w, &party[m])
}

// drawInventoryPanel (WPCVW b1e5) draws member m's armour class, equipment icons and
// inventory on the right of the sheet.
func drawInventoryPanel(w *Window, m int) {
	p := &party[m]
	mod := int(int8(p.AC[1])) + getLoadCategory(m).AC
	setCursor(w, 21, 5)
	wpcvwPrintMsg(217, w, 15)
	w.CurX++
	printInt(w, int(int8(p.AC[0]))+mod, 2, 4)
	printString(w, wpcvwACOpen, 15)
	if mod >= 0 {
		c := 4
		if mod == 0 {
			c = 9
		}
		putChar(w, '+', c)
		printInt(w, mod, 1, c)
	} else {
		printInt(w, mod, 2, 6)
	}
	putChar(w, ')', 15)
	setCursor(w, 21, 6)
	for i := 0; i < 6; i++ {
		putCharHi(w, ' ', 3)
		icon := 120
		if e := p.Equipped[2+i]; e != 255 {
			icon = int(p.Items[e].Icon)
		} else if i != 0 {
			icon = i + 120
		}
		putCharHi(w, byte(icon+1), 4)
		if i < 5 {
			putCharHi(w, 28, 1)
		} else {
			putCharHi(w, ' ', 3)
		}
	}
	c := 11
	for i := 0; i < 6; i++ {
		add := 0
		if i != 0 {
			add = int(int8(p.AC[2])) + mod
		}
		setCursor(w, i*3+21, 7)
		printInt(w, int(int8(p.AC[2+i]))+add, 3, (c+8)%16)
		c = 15
	}
	for i := 0; i < int(p.ItemCount); i++ {
		if !pvNamesLoaded {
			rec := itemRec(int(p.Items[i].ID))
			pvItemNames[i] = cstr(rec.Name[:])
		}
		mark, color := byte(' '), 9
		for s := 0; s < 8; s++ {
			if int(p.Equipped[s]) == i {
				mark = 23
				switch {
				case s < 2:
					color = 6
				case s == 2:
					color = 3
				default:
					color = 7
				}
			}
		}
		setCursor(w, 21, i+9)
		putChar(w, mark, 4)
		pvItemRowColor[i] = color
		printString(w, pvItemNames[i], color)
		padToColumn(w, 34, 0)
		if stackable[p.Items[i].Type] == 1 {
			setCursor(w, 34, i+9)
			printInt(w, int(p.Items[i].Qty), 3, 1)
		} else {
			padToColumn(w, 37, 0)
		}
		setCursor(w, 38, i+9)
		putCharHi(w, p.Items[i].Icon+1, 4)
	}
	if p.ItemCount < 10 {
		clearWindowArea(w, 21, int(p.ItemCount)+9, 36, 18)
	}
	for i := int(p.ItemCount); i < 10; i++ {
		setCursor(w, 38, i+9)
		putCharHi(w, ' ', 3)
	}
	pvNamesLoaded = true
}

// drawSheet (WPCVW 93f8) redraws member m's whole sheet.
func drawSheet(sheet, menu *Window, m int) {
	fillWindow(sheet, 0, 1)
	fillWindow(menu, ' ', 3)
	deferWindowDraw(sheet)
	drawSheetFrame(sheet)
	drawStatsPanel(sheet, m)
	drawInventoryPanel(sheet, m)
	resumeWindowDraw(sheet)
	refreshAllWindows()
}

// closeSheet (WPCVW a0ae) removes the sheet's windows and restores the bottom line for
// the screen the sheet returns to.
func closeSheet(menu, sheet *Window) {
	destroyWindow(menu, false)
	destroyWindow(sheet, false)
	switch returnCmd {
	case 5:
		eraseCursorMark(bottomLineWin, false)
	case 4:
		eraseWindow(bottomLineWin, false)
	case 13:
		eraseCursorMark(bottomLineWin, false)
		deferWindowDraw(bottomLineWin)
	}
}

// isMemberAbleQuiet (is_member_able, WPCVW a075) is whether member m is fine and neither
// asleep nor paralysed.
func isMemberAbleQuiet(m int) bool {
	p := &party[m]
	return p.Condition == 0 && p.Status[2] == 0 && p.Status[3] == 0
}

// wpcvwViewCharacter (view_character, WPCVW ad5a) runs member m's character sheet until
// Exit; returnCmd says where from (4 the start menu, 13 combat, else the maze).
func wpcvwViewCharacter(m int) {
	gameState = int(returnCmd)
	menuKeys = getMessage(300)
	pvCardWin = createWindow(20, 4, 20, 16, 26, -1, 0, false)
	fillWindow(pvCardWin, ' ', 3)
	deferWindowDraw(pvCardWin)
	drawPortraitFrame(pvCardWin)
	sheet := createWindow(0, 0, 40, 20, 20, -1, 0, false)
	menu := createWindow(0, 20, 40, 4, 15, -1, 0, false)
	resumeWindowDraw(bottomLineWin)
	eraseWindow(bottomLineWin, false)
	pvNamesLoaded = false
	for {
		drawSheet(sheet, menu, m)
		canCast := true
		// Entries: equip, cast, trade, inspect, swag, merge, use, drop, skills, modify,
		// other member, exit.
		enabled := []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, -1}
		switch returnCmd {
		case 4:
			for i := 0; i < 11; i++ {
				enabled[i] = 0
			}
			enabled[0], enabled[3], enabled[8], enabled[10], enabled[1], enabled[4] = 1, 1, 1, 1, 1, 1
			canCast = false
		case 13:
			for i := 0; i < 11; i++ {
				enabled[i] = 0
			}
			enabled[0], enabled[7], enabled[3], enabled[8], enabled[1], enabled[4] = 1, 1, 1, 1, 1, 1
			canCast = false
		}
		p := &party[m]
		if partyCount < 2 {
			enabled[10], enabled[2] = 0, 0
		}
		if p.ItemCount == 0 {
			enabled[3], enabled[7], enabled[6] = 0, 0, 0
		}
		if p.ItemCount < 2 {
			enabled[5] = 0
		}
		if p.Gold == 0 && p.ItemCount == 0 {
			enabled[2] = 0
		}
		if !isMemberAbleQuiet(m) {
			enabled[1], enabled[6] = 0, 0
		}
		flushKeys()
		choice := wpcvwRunMenu(menu, 301, enabled, 2, 1, 6, 2, 5, wpcvwCountEnabledBefore(enabled, 11))
		switch choice {
		case 0:
			doEquipItems(sheet, menu, m)
		case 1:
			doCastMenu(sheet, menu, m, canCast)
		case 2:
			doTradeItems(sheet, menu, m)
		case 3:
			if pvChosenItem = chooseItem(sheet, menu, 450, m, false); pvChosenItem != -1 {
				showItemInfo(menu, m, pvChosenItem, 0)
			}
		case 7:
			if pvChosenItem = chooseItem(sheet, menu, 402, m, false); pvChosenItem != -1 {
				if p.Items[pvChosenItem].Flags&(1|2|64) != 0 {
					playSound(0)
				} else {
					wpcvwRemoveItem(m, pvChosenItem)
				}
			}
		case 6:
			if pvChosenItem = chooseItem(sheet, menu, 400, m, false); pvChosenItem != -1 {
				wpcvwDoUseItem(sheet, menu, m, pvChosenItem)
			}
		case 5:
			doMergeItems(sheet, menu, m)
		case 9:
			doModifyMenu(sheet, menu, m)
		case 4:
			doSwagMenu(sheet, menu, m)
		case 8:
			doSkillsMenu(menu, m)
		case 10:
			if c := chooseMember(menu, 500, m); c != -1 {
				curMember = int16(c)
				m = c
				pvNamesLoaded = false
			} else {
				curMember = int16(m)
			}
		}
		if choice == 11 {
			break
		}
	}
	destroyWindow(pvCardWin, false)
	closeSheet(menu, sheet)
}

// drawPortraitFrame (WPCVW 9fa8) draws the right-hand panel's frame.
func drawPortraitFrame(w *Window) {
	wpcvwPutCharC1(w, 0, 0, 20)
	wpcvwPutCharC1(w, 19, 0, 8)
	for i := 1; i < 19; i++ {
		wpcvwPutCharC1(w, i, 0, 7)
	}
	for i := 1; i < 15; i++ {
		wpcvwPutCharC1(w, 0, i, 5)
	}
	wpcvwPutCharC1(w, 0, 9, 10)
	wpcvwPutCharC1(w, 0, 11, 10)
	wpcvwPutCharC1(w, 0, 13, 10)
	wpcvwPutCharC1(w, 0, 15, 8)
}

// drawMemberNameRight (draw_member_name, WPCVW 8937) reprints member m's name plain in
// chooseMember's list.
func drawMemberNameRight(w *Window, m int) {
	setCursor(w, m%2*9+22, m/2+1)
	printStringHi(w, memberName(m), 3)
}

// chooseMember (WPCVW 897d) lets the player pick a member by name under title message
// title, sel first (-1 the cancel entry); -1 for cancel. With one member it is member 0.
func chooseMember(w *Window, title, sel int) int {
	if partyCount < 2 {
		return 0
	}
	deferWindowDraw(pvCardWin)
	for i := 0; i < partyCount; i++ {
		drawMemberCard(pvCardWin, i)
	}
	resumeWindowDraw(pvCardWin)
	fillWindow(w, ' ', 3)
	refreshAllWindows()
	s := getMessage(title)
	setCursor(w, 10-(len(s)+1)/2, 1)
	printStringHi(w, s, 3)
	const cx, cy = 8, 3
	cancel := getMessage(503)
	setCursor(w, cx, cy)
	if !mousePresent && sel == -1 {
		printString(w, cancel, -5)
	} else {
		printStringHi(w, cancel, 3)
	}
	for i := 0; i < partyCount; i++ {
		x, y := i%2*9+22, i/2+1
		setCursor(w, x, y)
		if !mousePresent && i == sel {
			printString(w, memberName(i), -5)
		} else {
			printStringHi(w, memberName(i), 3)
		}
		setRegion(i, int(w.X)+x, int(w.Y)+y, len(memberName(i))+int(w.X)+x-1, int(w.Y)+y)
	}
	setRegion(partyCount, int(w.X)+cx, int(w.Y)+cy, len(cancel)+int(w.X)+cx-1, int(w.Y)+cy)
	endRegions(partyCount + 1)
	for {
		if !mousePresent {
			if sel == -1 {
				setCursor(w, cx, cy)
				printString(w, cancel, -5)
			} else {
				setCursor(w, sel%2*9+22, sel/2+1)
				printString(w, memberName(sel), -5)
			}
		}
		k := wpcvwGetMenuInput()
		if menuClicked == 1 {
			sel = -1
			if k < partyCount {
				sel = k
			}
			break
		}
		switch k {
		case 1:
			if sel != -1 {
				drawMemberNameRight(w, sel)
				if sel%2 > 0 {
					sel--
				} else {
					sel = -1
				}
			}
		case 2:
			if sel != -1 && sel-2 >= 0 {
				drawMemberNameRight(w, sel)
				sel -= 2
			}
		case 3:
			if sel == -1 {
				setCursor(w, cx, cy)
				printStringHi(w, cancel, 3)
				sel = 0
			} else if sel%2 < 1 && sel+1 < partyCount {
				drawMemberNameRight(w, sel)
				sel++
			}
		case 4:
			if sel != -1 && sel+2 < partyCount {
				drawMemberNameRight(w, sel)
				sel += 2
			}
		}
		if k == 5 {
			break
		}
	}
	fillWindow(w, ' ', 3)
	deferWindowDraw(pvCardWin)
	return sel
}
