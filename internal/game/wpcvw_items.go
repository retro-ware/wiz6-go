package game

import "strconv"

// WPCVW: items: choosing, removing, the swag bag, trading, merging, using and inspecting.

// clearHandAttack (WPCVW 5c24) resets hand h of member m to the bare-handed attack.
func clearHandAttack(m, h int) {
	p := &party[m]
	hd := &p.Hands[h]
	hd.B0D, hd.B0C = 0, 0
	hd.Data = [8]uint8{}
	hd.ToHit = 0
	hd.Damage = Dice{Base: 0, Count: 2, Sides: 2}
	hd.Flags = 32
	hand := int(p.Skills[9]) / 5
	kiri := int(p.Skills[29]) / 5
	if p.Class == 13 || p.Class == 12 {
		hd.Damage.Sides = uint8(hand/3 + 2)
		hd.Flags |= 64
		hd.Data[6] = uint8(kiri + 5)
		return
	}
	if hand > 0 {
		hd.Damage.Sides = uint8(hand/4 + 1)
		hd.Flags |= 64
	}
	if kiri > 0 {
		hd.Data[6] = uint8(kiri)
	}
}

// wpcvwRemoveItem (remove_item, WPCVW 5d5b) removes inventory item i of member m,
// unequipping it and moving the rest up.
func wpcvwRemoveItem(m, i int) {
	p := &party[m]
	for s := 0; s < 8; s++ {
		if int(p.Equipped[s]) != i {
			continue
		}
		switch s {
		case 0, 1:
			clearHandAttack(m, s)
		case 2:
			p.AC[2] = 0
		default:
			p.AC[s] = p.AC[0]
		}
		p.Equipped[s] = 255
	}
	p.Items[i].ID = 0
	p.ItemCount--
	for j := i; j < int(p.ItemCount); j++ {
		p.Items[j] = p.Items[j+1]
		pvItemNames[j] = pvItemNames[j+1]
		pvItemRowColor[j] = pvItemRowColor[j+1]
		for s := 0; s < 8; s++ {
			if int(p.Equipped[s]) == j+1 {
				p.Equipped[s] = uint8(j)
			}
		}
	}
}

// removeSwagItem (WPCVW 5ec7) removes swag item i of member m.
func removeSwagItem(m, i int) {
	p := &party[m]
	p.SwagCount--
	for j := i; j < int(p.SwagCount); j++ {
		wpcvwSetItem(p, j+10, wpcvwGetItem(p, j+11))
		pvSwagNames[j] = pvSwagNames[j+1]
	}
}

// wpcvwUseItemCharge (use_item_charge, WPCVW 5f53) uses a charge of item i of member m,
// removing it when the last is gone.
func wpcvwUseItemCharge(m, i int) {
	it := &party[m].Items[i]
	if it.Qty > 0 {
		if it.Qty--; it.Qty == 0 {
			wpcvwRemoveItem(m, i)
		}
	}
}

// unhighlightItemRow (WPCVW d331) reprints an inventory or swag row plain.
func unhighlightItemRow(sheet *Window, row, color int, swag bool) {
	if !swag {
		setCursor(sheet, 22, row+9)
		printString(sheet, pvItemNames[row], color)
		return
	}
	setCursor(sheet, 2, row+3)
	printString(sheet, pvSwagNames[row], 9)
}

// chooseItem (WPCVW 5fac) lets the player pick an inventory (or swag) item of member m
// on sheet, with title message title in w; -1 for cancel.
func chooseItem(sheet, w *Window, title, m int, swag bool) int {
	p := &party[m]
	count := int(p.SwagCount)
	if !swag {
		drawInventoryPanel(sheet, m)
		count = int(p.ItemCount)
	}
	fillWindow(w, ' ', 3, true)
	refreshWindow(w)
	t, cancel := getMessage(title), getMessage(401)
	setCursor(w, 2, 1)
	printStringHi(w, t, 3)
	w.CurX++
	cx, cy := int(w.CurX), int(w.CurY)
	if mousePresent {
		printStringHi(w, cancel, 3)
	} else {
		printString(w, cancel, -5)
	}
	sel := -1
	sx, sy := int(sheet.X), int(sheet.Y)
	for i := 0; i < count; i++ {
		if !swag {
			setRegion(i, sx+22, sy+i+9, sx+37, sy+i+9)
		} else {
			setRegion(i, sx+2, sy+i+3, sx+17, sy+i+3)
		}
	}
	setRegion(count, int(w.X)+cx, int(w.Y)+cy, len(cancel)+int(w.X)+cx-1, int(w.Y)+cy)
	endRegions(count + 1)
	for {
		if !mousePresent {
			switch {
			case sel == -1:
				setCursor(w, cx, cy)
				printString(w, cancel, -5)
			case !swag:
				setCursor(sheet, 22, sel+9)
				printString(sheet, pvItemNames[sel], -5)
			default:
				setCursor(sheet, 2, sel+3)
				printString(sheet, pvSwagNames[sel], -5)
			}
		}
		k := wpcvwGetMenuInput()
		if menuClicked == 1 {
			sel = -1
			if k < count {
				sel = k
			}
			break
		}
		switch k {
		case 2, 4:
			if sel == -1 {
				setCursor(w, cx, cy)
				printStringHi(w, cancel, 3)
				sel = 0
				break
			}
			unhighlightItemRow(sheet, sel, pvItemRowColor[sel], swag)
			if k == 2 {
				if sel--; sel < 0 {
					sel = -1
				}
			} else if sel++; sel >= count {
				sel = -1
			}
		}
		if k == 5 {
			break
		}
	}
	if !swag {
		drawInventoryPanel(sheet, m)
	}
	return sel
}

// doSwagMenu (WPCVW 6317) runs the swag bag of member m: put in, take out, drop.
func doSwagMenu(sheet, menu *Window, m int) {
	p := &party[m]
	loaded := false
	w := createWindow(20, 4, 20, 16, 25, -1, 0, false)
	for {
		fillWindow(w, 0, 1, true)
		f := func(x, y int, ch byte) { wpcvwPutFrameChar(w, x, y, ch) }
		f(0, 0, 14)
		f(19, 0, 10)
		f(0, 15, 11)
		f(19, 15, 8)
		for i := 1; i < 19; i++ {
			f(i, 0, 12)
			f(i, 2, 12)
			f(i, 15, 7)
		}
		for i := 1; i < 15; i++ {
			f(0, i, 13)
			f(19, i, 5)
		}
		f(0, 2, 33)
		f(0, 9, 15)
		f(0, 11, 15)
		f(0, 13, 15)
		f(19, 2, 10)
		f(17, 2, 14)
		f(17, 15, 11)
		for i := 3; i < 15; i++ {
			f(17, i, 13)
		}
		deferWindowDraw(w)
		setCursor(w, 1, 1)
		printMessage(750, w, 18)
		wpcvwPutIconAt(w, 2, 1, 100)
		wpcvwPutIconAt(w, 17, 1, 100)
		for i := 0; i < int(p.SwagCount); i++ {
			it := wpcvwGetItem(p, i+10)
			if !loaded {
				rec := itemRec(int(it.ID))
				pvSwagNames[i] = cstr(rec.Name[:])
			}
			setCursor(w, 2, i+3)
			printString(w, pvSwagNames[i], 9)
			padToColumn(w, 14, 0)
			if stackable[it.Type] == 1 {
				setCursor(w, 14, i+3)
				printInt(w, int(it.Qty), 3, 1)
			} else {
				padToColumn(w, 17, 0)
			}
			setCursor(w, 18, i+3)
			putCharHi(w, it.Icon+1, 4)
		}
		if p.SwagCount < 12 {
			clearWindowArea(w, 1, int(p.SwagCount)+3, 16, 14)
		}
		for i := int(p.SwagCount); i < 12; i++ {
			setCursor(w, 18, i+3)
			putCharHi(w, ' ', 3)
		}
		loaded = true
		resumeWindowDraw(w)
		fillWindow(menu, ' ', 3, true)
		refreshWindow(w)
		refreshWindow(menu)
		// Entries: put in, take out, drop, exit.
		enabled := []int{1, 1, 1, 1, -1}
		if !(p.SwagCount < 12 && p.ItemCount != 0) {
			enabled[0] = 0
		}
		if !(p.SwagCount != 0 && p.ItemCount < 10) {
			enabled[1] = 0
		}
		if p.SwagCount == 0 {
			enabled[2] = 0
		}
		fillWindow(menu, ' ', 3, true)
		refreshWindow(menu)
		switch wpcvwRunMenu(menu, 751, enabled, 2, 1, 8, 2, 5, wpcvwCountEnabledBefore(enabled, 3)) {
		case 0:
			deferWindowDraw(w)
			fillWindow(menu, ' ', 3, true)
			refreshAllWindows()
			pvChosenItem = chooseItem(sheet, menu, 760, m, false)
			if pvChosenItem != -1 {
				if p.Items[pvChosenItem].Flags&1 != 0 {
					playSound(0)
				} else {
					wpcvwSetItem(p, int(p.SwagCount)+10, p.Items[pvChosenItem])
					pvSwagNames[p.SwagCount] = pvItemNames[pvChosenItem]
					p.SwagCount++
					wpcvwRemoveItem(m, pvChosenItem)
					deferWindowDraw(sheet)
					drawInventoryPanel(sheet, m)
					resumeWindowDraw(sheet)
				}
			}
			resumeWindowDraw(w)
		case 1:
			if i := chooseItem(w, menu, 761, m, true); i != -1 {
				p.Items[p.ItemCount] = wpcvwGetItem(p, i+10)
				pvItemNames[p.ItemCount] = pvSwagNames[i]
				p.ItemCount++
				removeSwagItem(m, i)
				drawInventoryPanel(sheet, m)
			}
		case 2:
			if i := chooseItem(w, menu, 762, m, true); i != -1 {
				if wpcvwGetItem(p, i+10).Flags&64 != 0 {
					playSound(0)
				} else {
					removeSwagItem(m, i)
				}
			}
		case 3:
			fillWindow(menu, ' ', 3, true)
			destroyWindow(w, true)
			return
		}
	}
}

// doTradeItems (WPCVW 96a2) gives gold and items of member m to another member.
func doTradeItems(sheet, menu *Window, m int) {
	to := chooseMember(menu, 501, -1)
	if to == -1 || to == m {
		return
	}
	p, q := &party[m], &party[to]
	if p.Gold > 0 {
		for {
			n := wpcvwInputNumber(menu, 404, 8)
			if n >= 0 && p.Gold >= n {
				p.Gold -= n
				q.Gold += n
				break
			}
			playSound(0)
		}
	}
	drawSheet(sheet, menu, m)
	for int(q.ItemCount)+int(q.SwagCount) < 22 && p.ItemCount > 0 {
		pvChosenItem = chooseItem(sheet, menu, 403, m, false)
		if pvChosenItem == -1 {
			return
		}
		it := &p.Items[pvChosenItem]
		whole, give := true, true
		n := int(it.Qty)
		if it.Flags&1 != 0 || it.Flags&2 != 0 {
			whole, give = false, false
			playSound(0)
		} else if stackable[it.Type] == 1 {
			for {
				n = int(int16(wpcvwInputNumber(menu, 405, 3)))
				if int(it.Qty) < n {
					n = -1
					playSound(0)
				}
				if n != -1 {
					break
				}
			}
			if int(it.Qty) > n {
				whole = false
			}
			if n == 0 {
				give = false
			}
		}
		if !give {
			continue
		}
		// Stack onto a like item of the receiver's inventory, else its swag, else add.
		add, slot := 1, int(q.ItemCount)
		stacks := func(s int) bool {
			o := wpcvwGetItem(q, s)
			return o.ID == it.ID && stackable[o.Type] == 1 && int(o.Qty)+n <= 250
		}
		for s := slot - 1; s >= 0; s-- {
			if stacks(s) {
				slot, add = s, 0
			}
		}
		if slot >= 10 {
			add, slot = 2, int(q.SwagCount)+10
			for s := int(q.SwagCount) - 1; s >= 0; s-- {
				if stacks(s + 10) {
					slot, add = s+10, 0
				}
			}
		}
		o := wpcvwGetItem(q, slot)
		if add > 0 {
			o = *it
			o.Qty = uint8(n)
			o.Flags &^= 1
			if add == 1 {
				q.ItemCount++
			} else {
				q.SwagCount++
			}
		} else {
			o.Qty += uint8(n)
		}
		wpcvwSetItem(q, slot, o)
		if whole {
			wpcvwRemoveItem(m, pvChosenItem)
		} else {
			it.Qty -= uint8(n)
		}
		drawSheet(sheet, menu, m)
	}
}

// replaceItem (WPCVW 9b3f) turns item i of member m into item id (a recipe's result) and
// removes item other.
func replaceItem(sheet *Window, m, id, i, other int) {
	p := &party[m]
	rec := itemRec(id)
	pvItemNames[i] = cstr(rec.Name[:])
	it := &p.Items[i]
	it.ID = int16(id)
	it.Icon = rec.Icon
	it.Type = rec.Type
	it.Weight = rec.Weight
	it.Qty = uint8(rollDice(rec.Charges))
	wpcvwRemoveItem(m, other)
	deferWindowDraw(sheet)
	drawInventoryPanel(sheet, m)
	resumeWindowDraw(sheet)
	refreshAllWindows()
	if i > other {
		i--
	}
	setCursor(sheet, 22, i+9)
	printString(sheet, pvItemNames[i], -5)
	padToColumn(sheet, 37, 0)
	setCursor(sheet, 38, i+9)
	putCharHi(sheet, p.Items[i].Icon+1, 4)
	playSound(0)
	waitSecondsOrEnter(messageDelay)
}

// tryCombineRecipe (WPCVW 9c8e) combines items a and b of member m if a recipe takes
// them.
func tryCombineRecipe(sheet *Window, m, a, b int) bool {
	ia, ib := party[m].Items[a].ID, party[m].Items[b].ID
	for _, r := range recipes {
		if ia == r[0] && ib == r[1] || ia == r[1] && ib == r[0] {
			replaceItem(sheet, m, int(r[2]), a, b)
			return true
		}
	}
	return false
}

// doMergeItems (WPCVW 9d8a) merges two items of member m: stacks of the same item, or a
// recipe.
func doMergeItems(sheet, menu *Window, m int) {
	p := &party[m]
	if p.ItemCount < 2 {
		return
	}
	if pvChosenItem = chooseItem(sheet, menu, 406, m, false); pvChosenItem == -1 {
		return
	}
	b := chooseItem(sheet, menu, 407, m, false)
	if b == -1 {
		return
	}
	if b == pvChosenItem {
		playSound(0)
		return
	}
	if p.Items[b].Flags&1 != 0 {
		pvChosenItem, b = b, pvChosenItem
		if p.Items[b].Flags&1 != 0 {
			playSound(0)
			return
		}
	}
	a := pvChosenItem
	if p.Items[a].ID == p.Items[b].ID && stackable[p.Items[a].Type] == 1 {
		if n := int(p.Items[a].Qty) + int(p.Items[b].Qty); n <= 250 {
			p.Items[a].Qty = uint8(n)
			wpcvwRemoveItem(m, b)
		} else {
			p.Items[a].Qty = 250
			p.Items[b].Qty = uint8(n - 250)
		}
	} else if !tryCombineRecipe(sheet, m, a, b) {
		playSound(0)
	}
	drawSheet(sheet, menu, m)
}

// wpcvwDoUseItem (do_use_item, WPCVW 8fbf) uses item i of member m: the item record's
// Unk69 picks a reading or key action, else Unk28 is a spell it casts.
func wpcvwDoUseItem(sheet, menu *Window, m, i int) {
	p := &party[m]
	rec := itemRec(int(p.Items[i].ID))
	if rec.Unk69 > 0 {
		pickedIs := func(id int16) bool {
			j := chooseItem(sheet, menu, 408, m, false)
			return j >= 0 && p.Items[j].ID == id // the C reads slot -1 on cancel
		}
		switch rec.Unk69 {
		case 1:
			j := chooseItem(sheet, menu, 408, m, false)
			if j >= 0 && p.Items[j].ID == 395 {
				wpcvwShowTextPages(menu, 11350)
			} else if j >= 0 && p.Items[j].ID == 403 {
				wpcvwShowTextPages(menu, 12200)
			}
		case 2:
			wpcvwShowTextPages(menu, int(int16(rec.Price))) // Price's low word is the text
		case 3: // a spell book: Unk28 is the spell
			if testBit(mask16(rec.ClassMask), int(p.Class)) {
				s := int(rec.Unk28) % 256
				if int(spellTable[s].Level)<<1 > int(p.Level) {
					wpcvwShowTextPages(menu, 725)
				} else {
					setBit(p.KnownSpells[:], s, true)
					wpcvwShowTextPages(menu, 720)
					wpcvwRemoveItem(m, i)
				}
			} else {
				wpcvwShowTextPages(menu, 730)
			}
		case 4: // on map Damage.Base, clear event flag Damage.Count and set Damage.Sides
			if mp := int(rec.Damage.Base); gameStateArg == mp {
				setBit(levelFlags[mp][:], int(rec.Damage.Count), false)
				setBit(hdr.LevelFlags[mp][:], int(rec.Damage.Count), false)
				setBit(levelFlags[mp][:], int(rec.Damage.Sides), true)
				setBit(hdr.LevelFlags[mp][:], int(rec.Damage.Sides), true)
				wpcvwUseItemCharge(m, i)
			}
		case 5:
			if gameStateArg == 8 {
				partyCounters[0].B = 1
			}
		case 6:
			if pickedIs(388) {
				wpcvwShowTextPages(menu, 18520)
			}
		}
		return
	}
	if rec.Unk28 == 0 {
		return
	}
	power, s := int(rec.Unk28)/256, int(rec.Unk28)%256
	if spellTable[s].Target == 1 {
		if pvSpellTarget = chooseMember(menu, 504, -1); pvSpellTarget == -1 {
			return
		}
	}
	castSpell(m, s, power, pvSpellTarget, sheet, menu)
	wpcvwUseItemCharge(m, i)
}

// printCipher (WPCVW b659) prints the 4-letter code of v with the letters of message 479.
func printCipher(w *Window, v, color int) {
	key := getMessage(479)
	for i := 0; i < 4; i++ {
		d := v % 10
		d = (i*d + d) % 10
		putChar(w, byte(byteAt(key, d)), color)
		v = v/10 - d + 10
	}
}

// showItemInfo (WPCVW b6c4) shows item i of member m in a window: unidentified (power 0)
// its kind, who may use it, its attack flags, hands and weight; identified with spell
// power, its damage, bonuses, powers and resistances, if power*3 beats its Unk62.
func showItemInfo(menu *Window, m, i, power int) {
	p := &party[m]
	rec := itemRec(int(p.Items[i].ID))
	w := createWindow(20, 8, 20, 12, 25, -1, 0, false)
	fillWindow(w, 0, 1, true)
	f := func(x, y int, ch byte) { wpcvwPutFrameChar(w, x, y, ch) }
	f(0, 0, 21)
	f(19, 0, 8)
	f(0, 11, 11)
	f(19, 11, 8)
	for x := 1; x < 19; x++ {
		f(x, 0, 7)
		f(x, 2, 18)
		f(x, 11, 7)
	}
	for y := 2; y < 11; y++ {
		f(0, y, 13)
		f(19, y, 5)
	}
	f(0, 2, 36)
	f(19, 2, 19)
	f(0, 5, 15)
	f(0, 7, 15)
	f(0, 9, 15)
	f(0, 1, 5)
	deferWindowDraw(w)
	setCursor(w, 1, 1)
	for x := 1; x < 20; x++ {
		putCharHi(w, ' ', 3)
	}
	wpcvwPutIconAt(w, 1, 1, rec.Icon+1)
	wpcvwPutIconAt(w, 18, 1, rec.Icon+1)
	name := cstr(rec.Name[:])
	setCursor(w, 10-(len(name)+1)/2, 1)
	printStringHi(w, name, 3)
	centered := func(s string, row, color int) {
		setCursor(w, 10-(len(s)+1)/2, row)
		printString(w, s, color)
	}

	// What the item's type shows (l122[6..19]).
	var (
		specials, regen, damage, masks, spell, resists bool
		attackFlags, skill, typeName, hands, acLine    bool
		primary, secondary                             bool
		acColor                                        = 7
	)
	masks = true
	switch rec.Type {
	case 0, 1, 2, 3:
		primary, hands, skill, attackFlags, damage, specials, regen = true, true, true, true, true, true, true
	case 4:
		secondary, damage, specials = true, true, true
	case 5:
		acLine, acColor, resists, regen = true, 3, true, true
	case 6, 7, 8, 9, 10:
		acLine, typeName, resists, regen = true, true, true, true
	case 11:
		acLine, resists, regen = true, true, true
	case 12, 16:
		masks, primary, spell, attackFlags = false, true, true, true
	case 13:
		masks, secondary, spell = false, true, true
	case 14, 15:
		secondary, attackFlags, regen = true, true, true
	}

	switch {
	case power == 0:
		centered(getMessage(int(rec.Type)+1550), 3, 1)
		if masks {
			setCursor(w, 3, 4)
			s := getMessage(454)
			for b := 0; b < 11; b++ {
				putChar(w, byte(byteAt(s, b)), maskColor(testBit(mask16(rec.RaceMask), b)))
			}
			w.CurX++
			s = getMessage(455)
			for b := 0; b < 2; b++ {
				putChar(w, byte(byteAt(s, b)), maskColor(testBit([]byte{rec.SexMask, rec.Flags}, b)))
			}
			setCursor(w, 3, 5)
			s = getMessage(456)
			for b := 0; b < 14; b++ {
				putChar(w, byte(byteAt(s, b)), maskColor(testBit(mask16(rec.ClassMask), b)))
			}
		} else if spell && rec.Unk28 != 0 {
			centered(getMessage(int(rec.Unk28)%256+4000), 5, 4)
		}
		if skill {
			s, c := getMessage(1567), 9
			if testBit(mask16(rec.RaceMask), int(p.Race)) && testBit(mask16(rec.ClassMask), int(p.Class)) && testBit([]byte{rec.SexMask, rec.Flags}, int(p.Sex)) {
				s, c = getMessage(weaponSkill(rec.Icon)+5500), 2
			}
			centered(s, 6, c)
		}
		if attackFlags {
			if rec.Type == 3 {
				setCursor(w, 3, 7)
				printString(w, getMessage(3058), 5)
			} else {
				n := 0
				for b := 0; b < 8; b++ {
					if testBit([]byte{rec.HandFlags, rec.Unk73}, b) {
						setCursor(w, n/3<<3+3, n%3+7)
						printString(w, getMessage(b+3050), 5)
						n++
					}
				}
			}
		}
		setCursor(w, 1, 10)
		if primary || secondary {
			s := getMessage(451)
			putChar(w, byte(byteAt(s, 0)), map[bool]int{true: 6, false: 14}[primary])
			putChar(w, '/', 14)
			putChar(w, byte(byteAt(s, 1)), map[bool]int{true: 6, false: 14}[rec.Flags&8 != 0])
		} else if acLine {
			printString(w, getMessage(452), acColor+8)
			w.CurX++
			printString(w, strconv.Itoa(-int(int8(rec.AC))), acColor)
		}
		if hands {
			msg := 457
			if rec.Flags&4 != 0 {
				msg = 458
			}
			setCursor(w, 8, 10)
			printString(w, getMessage(msg), 4)
		} else if typeName {
			setCursor(w, 8, 10)
			printString(w, getMessage(int(rec.Type)+454), 4)
		}
		setCursor(w, 14, 10)
		printInt(w, int(rec.Weight)/10, 3, 9)
		putChar(w, '.', 9)
		printInt(w, int(rec.Weight)%10, 1, 9)

	case power*3 > int(rec.Unk62):
		row := 3
		if damage {
			setCursor(w, 1, 3)
			printString(w, getMessage(466), 14)
			a, b, c := strconv.Itoa(int(rec.Damage.Count)), strconv.Itoa(int(rec.Damage.Sides)), strconv.Itoa(int(rec.Damage.Base))
			setCursor(w, 19-(len(a)+len(b)+len(c)+2), 3)
			printString(w, a, 5)
			printString(w, getMessage(478), 5)
			printString(w, b, 5)
			putChar(w, '+', 5)
			printString(w, c, 5)
			if th := int8(rec.ToHitPen); th != 0 {
				setCursor(w, 1, 4)
				printString(w, getMessage(467), 14)
				setCursor(w, 7, 4)
				if th >= 0 {
					putChar(w, '+', 5)
				}
				printString(w, strconv.Itoa(int(th)), 5)
			}
			row = 5
		}
		if d := int(rec.Regen) - int(rec.Drain); regen && d != 0 {
			row = 3
			if damage {
				row = 4
			}
			setCursor(w, 11, row)
			printString(w, getMessage(468), 14)
			setCursor(w, 17, row)
			if d >= 0 {
				putChar(w, '+', 5)
			}
			printString(w, strconv.Itoa(d), 5)
			row = 5
		}
		if rec.Unk63 != 0 {
			setCursor(w, 1, row)
			printString(w, getMessage(465), 3)
			setCursor(w, 15, row)
			printCipher(w, int(rec.Unk63), 1)
			row++
		}
		if rec.Unk28 != 0 {
			setCursor(w, 1, row)
			printString(w, getMessage(int(rec.Unk28)%256+4000), 4)
			setCursor(w, 16, row)
			if p.Items[i].Qty == 0 {
				printString(w, getMessage(480), 1)
			} else {
				printInt(w, int(p.Items[i].Qty), 3, 1)
			}
			row++
		}
		if row > 3 && row < 8 {
			row++
		}
		if rec.Unk68 != 0 {
			setCursor(w, 1, row)
			printString(w, getMessage(469), 7)
			setCursor(w, 15, row)
			printCipher(w, int(rec.Unk68), 1)
			row++
		}
		if specials {
			n := 0
			for b := 0; b < 8; b++ {
				if rec.HandData[b] > 0 {
					setCursor(w, n/3<<3+3, n%3+row)
					printString(w, getMessage(b+470), 5)
					n++
				}
			}
		} else if resists {
			for r := 0; r < 6; r++ {
				v := min((int(rec.SkillBonus[2*r+1])+int(rec.SkillBonus[2*r+2]))/2, 99)
				setCursor(w, r/3*9+1, r%3+row)
				printString(w, getMessage(r+3950), realmColors[r])
				setCursor(w, r/3*9+7, r%3+row)
				printInt(w, v, 2, 1)
				putChar(w, '%', realmColors[r])
			}
		}

	default:
		s := getMessage(482)
		setCursor(w, 10-(len(s)+1)/2, 4)
		printString(w, s, 1)
	}
	resumeWindowDraw(w)
	fillWindow(menu, ' ', 3, true)
	deferWindowDraw(menu)
	s := getMessage(609)
	setCursor(menu, 20-(len(s)+1)/2, 1)
	printStringHi(menu, s, 3)
	resumeWindowDraw(menu)
	refreshWindow(w)
	refreshWindow(menu)
	wpcvwWaitEnterOrClick()
	destroyWindow(w, false)
}

// maskColor is the colour of a race, sex or class letter: 6 allowed, 11 not.
func maskColor(ok bool) int {
	if ok {
		return 6
	}
	return 11
}

// weaponSkill is the weapon skill (message 5500+) of a weapon icon (show_item_info's
// l124). The C leaves it unset for other icons; 0 here.
func weaponSkill(icon uint8) int {
	switch icon {
	case 0, 18, 89:
		return 0
	case 1, 14, 15, 16, 51, 53, 55, 58, 59:
		return 1
	case 2, 3:
		return 2
	case 4, 5, 6, 52, 54, 56, 98:
		return 3
	case 7, 8, 9, 10, 17, 19, 57, 90:
		return 4
	case 11, 12:
		return 7
	case 13:
		return 6
	case 24, 38:
		return 8
	case 20, 21:
		return 5
	}
	return 0
}
