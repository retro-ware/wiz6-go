package game

import "strconv"

// WPCVW: items: choosing, removing, the swag bag, trading, merging, using and inspecting.

// clearHandAttack (WPCVW 5c24) resets hand h of member m to the bare-handed attack.
func clearHandAttack(m, h int) {
	p := &party[m]
	hd := &p.Hands[h]
	hd.Range, hd.PoisonStr = 0, 0
	hd.HitEffects = [8]uint8{}
	hd.ToHit = 0
	hd.Damage = Dice{Base: 0, Count: bareHandDamage, Sides: bareHandDamage}
	hd.Flags = handPunch
	hand := int(p.Skills[skillUnarmed]) / 5
	kiri := int(p.Skills[skillKirijutsu]) / 5
	if p.Class == classNinja || p.Class == classMonk {
		hd.Damage.Sides = uint8(hand/3 + 2)
		hd.Flags |= handKick
		hd.HitEffects[handCritical] = uint8(kiri + 5)
		return
	}
	if hand > 0 {
		hd.Damage.Sides = uint8(hand/4 + 1)
		hd.Flags |= handKick
	}
	if kiri > 0 {
		hd.HitEffects[handCritical] = uint8(kiri)
	}
}

// removeItemShown (remove_item, WPCVW 5d5b) is removeItem for an inventory item that
// also moves the sheet's item names and row colours up with the items.
func removeItemShown(m, i int) {
	p := &party[m]
	for s := range equipSlots {
		if int(p.Equipped[s]) != i {
			continue
		}
		switch s {
		case equipPrimary, equipSecondary:
			clearHandAttack(m, s)
		case equipMisc:
			p.AC[equipMisc] = 0
		default:
			p.AC[s] = p.AC[acBase]
		}
		p.Equipped[s] = noEquippedItem
	}
	p.Items[i].ID = 0
	p.ItemCount--
	for j := i; j < int(p.ItemCount); j++ {
		p.Items[j] = p.Items[j+1]
		pvItemNames[j] = pvItemNames[j+1]
		pvItemRowColor[j] = pvItemRowColor[j+1]
		for s := range equipSlots {
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
		p.Items[packSlots+j] = p.Items[packSlots+j+1]
		pvSwagNames[j] = pvSwagNames[j+1]
	}
}

// useItemChargeShown (use_item_charge, WPCVW 5f53) is useItemCharge with
// removeItemShown.
func useItemChargeShown(m, i int) {
	it := &party[m].Items[i]
	if it.Qty > 0 {
		if it.Qty--; it.Qty == 0 {
			removeItemShown(m, i)
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
	if dtLog != nil { // stubbed in the difftest: the first item
		return 0
	}
	p := &party[m]
	count := int(p.SwagCount)
	if !swag {
		drawInventoryPanel(sheet, m)
		count = int(p.ItemCount)
	}
	fillWindow(w, ' ', 3)
	refreshWindow(w)
	t, cancel := getMessage(title), getMessage(msgNoItemCancel)
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
		k := getMenuInputBlink()
		if menuClicked == 1 {
			sel = -1
			if k < count {
				sel = k
			}
			break
		}
		switch k {
		case menuKeyUp, menuKeyDown:
			if sel == -1 {
				setCursor(w, cx, cy)
				printStringHi(w, cancel, 3)
				sel = 0
				break
			}
			unhighlightItemRow(sheet, sel, pvItemRowColor[sel], swag)
			if k == menuKeyUp {
				if sel--; sel < 0 {
					sel = -1
				}
			} else if sel++; sel >= count {
				sel = -1
			}
		}
		if k == menuKeyEnter {
			break
		}
	}
	if !swag {
		drawInventoryPanel(sheet, m)
	}
	return sel
}

// doSwagMenu's entries (messages 751+).
const (
	swagMenuAdd = iota
	swagMenuRemove
	swagMenuDrop
	swagMenuExit
)

// doSwagMenu (WPCVW 6317) runs the swag bag of member m: put in, take out, drop.
func doSwagMenu(sheet, menu *Window, m int) {
	p := &party[m]
	loaded := false
	w := createWindow(20, 4, 20, 16, 25, borderNone, 0, false)
	for {
		fillWindow(w, 0, 1)
		f := func(x, y int, ch byte) { setFrameGlyph(w, x, y, ch) }
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
		printMessage(msgSwagBag, w, 18)
		putIconAt(w, 2, 1, 100)
		putIconAt(w, 17, 1, 100)
		for i := 0; i < int(p.SwagCount); i++ {
			it := p.Items[packSlots+i]
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
		if p.SwagCount < swagSlots {
			clearWindowArea(w, 1, int(p.SwagCount)+3, 16, 14)
		}
		for i := int(p.SwagCount); i < swagSlots; i++ {
			setCursor(w, 18, i+3)
			putCharHi(w, ' ', 3)
		}
		loaded = true
		resumeWindowDraw(w)
		fillWindow(menu, ' ', 3)
		refreshWindow(w)
		refreshWindow(menu)
		enabled := []int{1, 1, 1, 1, -1}
		if !(p.SwagCount < swagSlots && p.ItemCount != 0) {
			enabled[swagMenuAdd] = 0
		}
		if !(p.SwagCount != 0 && p.ItemCount < packSlots) {
			enabled[swagMenuRemove] = 0
		}
		if p.SwagCount == 0 {
			enabled[swagMenuDrop] = 0
		}
		fillWindow(menu, ' ', 3)
		refreshWindow(menu)
		switch menuSelect(menu, msgSwagMenu, enabled, 2, 1, 8, 2, 5, countEnabledItems(enabled, swagMenuExit), getMenuInputBlink) {
		case swagMenuAdd:
			deferWindowDraw(w)
			fillWindow(menu, ' ', 3)
			refreshAllWindows()
			pvChosenItem = chooseItem(sheet, menu, msgPutWhichItemIn, m, false)
			if pvChosenItem != -1 {
				if p.Items[pvChosenItem].Flags&itemEquipped != 0 {
					playSound(soundBeep)
				} else {
					p.Items[packSlots+int(p.SwagCount)] = p.Items[pvChosenItem]
					pvSwagNames[p.SwagCount] = pvItemNames[pvChosenItem]
					p.SwagCount++
					removeItemShown(m, pvChosenItem)
					deferWindowDraw(sheet)
					drawInventoryPanel(sheet, m)
					resumeWindowDraw(sheet)
				}
			}
			resumeWindowDraw(w)
		case swagMenuRemove:
			if i := chooseItem(w, menu, msgRemoveWhichItem, m, true); i != -1 {
				p.Items[p.ItemCount] = p.Items[packSlots+i]
				pvItemNames[p.ItemCount] = pvSwagNames[i]
				p.ItemCount++
				removeSwagItem(m, i)
				drawInventoryPanel(sheet, m)
			}
		case swagMenuDrop:
			if i := chooseItem(w, menu, msgSwagDropWhichItem, m, true); i != -1 {
				if p.Items[packSlots+i].Flags&itemKept != 0 {
					playSound(soundBeep)
				} else {
					removeSwagItem(m, i)
				}
			}
		case swagMenuExit:
			fillWindow(menu, ' ', 3)
			destroyWindow(w, true)
			return
		}
	}
}

// doTradeItems (WPCVW 96a2) gives gold and items of member m to another member.
func doTradeItems(sheet, menu *Window, m int) {
	to := chooseMember(menu, msgTradeWithWho, -1)
	if to == -1 || to == m {
		return
	}
	p, q := &party[m], &party[to]
	if p.Gold > 0 {
		for {
			n := inputNumber(menu, msgTradeGoldAmount, 8)
			if n >= 0 && p.Gold >= n {
				p.Gold -= n
				q.Gold += n
				break
			}
			playSound(soundBeep)
		}
	}
	drawSheet(sheet, menu, m)
	for int(q.ItemCount)+int(q.SwagCount) < itemSlots && p.ItemCount > 0 {
		pvChosenItem = chooseItem(sheet, menu, msgTradeWhichItem, m, false)
		if pvChosenItem == -1 {
			return
		}
		it := &p.Items[pvChosenItem]
		whole, give := true, true
		n := int(it.Qty)
		if it.Flags&itemEquipped != 0 || it.Flags&itemCursed != 0 {
			whole, give = false, false
			playSound(soundBeep)
		} else if stackable[it.Type] == 1 {
			for {
				n = int(int16(inputNumber(menu, msgTradeHowManyAmount, 3)))
				if int(it.Qty) < n {
					n = -1
					playSound(soundBeep)
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
			o := q.Items[s]
			return o.ID == it.ID && stackable[o.Type] == 1 && int(o.Qty)+n <= maxStackQty
		}
		for s := slot - 1; s >= 0; s-- {
			if stacks(s) {
				slot, add = s, 0
			}
		}
		if slot >= packSlots {
			add, slot = 2, packSlots+int(q.SwagCount)
			for s := int(q.SwagCount) - 1; s >= 0; s-- {
				if stacks(packSlots + s) {
					slot, add = packSlots+s, 0
				}
			}
		}
		o := q.Items[slot]
		if add > 0 {
			o = *it
			o.Qty = uint8(n)
			o.Flags &^= itemEquipped
			if add == 1 {
				q.ItemCount++
			} else {
				q.SwagCount++
			}
		} else {
			o.Qty += uint8(n)
		}
		q.Items[slot] = o
		if whole {
			removeItemShown(m, pvChosenItem)
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
	removeItemShown(m, other)
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
	playSound(soundBeep)
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
	if pvChosenItem = chooseItem(sheet, menu, msgMergeWhichItem, m, false); pvChosenItem == -1 {
		return
	}
	b := chooseItem(sheet, menu, msgMergeWithItem, m, false)
	if b == -1 {
		return
	}
	if b == pvChosenItem {
		playSound(soundBeep)
		return
	}
	if p.Items[b].Flags&itemEquipped != 0 {
		pvChosenItem, b = b, pvChosenItem
		if p.Items[b].Flags&itemEquipped != 0 {
			playSound(soundBeep)
			return
		}
	}
	a := pvChosenItem
	if p.Items[a].ID == p.Items[b].ID && stackable[p.Items[a].Type] == 1 {
		if n := int(p.Items[a].Qty) + int(p.Items[b].Qty); n <= maxStackQty {
			p.Items[a].Qty = uint8(n)
			removeItemShown(m, b)
		} else {
			p.Items[a].Qty = maxStackQty
			p.Items[b].Qty = uint8(n - maxStackQty)
		}
	} else if !tryCombineRecipe(sheet, m, a, b) {
		playSound(soundBeep)
	}
	drawSheet(sheet, menu, m)
}

// ItemRec.UseAction values: what using an item does instead of casting its spell.
const (
	useReadWithItem    = 1 // pick an item to read (395, 403) with this one
	useShowText        = 2 // show the text pages of message Price
	useSpellBook       = 3 // learn the spell
	useSwitchMapFlags  = 4 // a key: switch two event flags of a map
	useDirectionOnMap8 = 5 // on map 8 set the direction effect's B
	useReadWithItem388 = 6 // pick item 388 to read with this one
)

// useItemOnSheet (do_use_item, WPCVW 8fbf) uses item i of member m: the item record's
// UseAction picks a reading or key action, else Spell is a spell it casts.
func useItemOnSheet(sheet, menu *Window, m, i int) {
	p := &party[m]
	rec := itemRec(int(p.Items[i].ID))
	if rec.UseAction > 0 {
		pickedIs := func(id int16) bool {
			j := chooseItem(sheet, menu, msgUseOnWhichItem, m, false)
			return j >= 0 && p.Items[j].ID == id // the C reads slot -1 on cancel
		}
		switch rec.UseAction {
		case useReadWithItem:
			j := chooseItem(sheet, menu, msgUseOnWhichItem, m, false)
			if j >= 0 && p.Items[j].ID == 395 {
				showTextPages(menu, msgDecoderRing)
			} else if j >= 0 && p.Items[j].ID == 403 {
				showTextPages(menu, msgDecoderRingText)
			}
		case useShowText:
			showTextPages(menu, int(int16(rec.Price))) // Price's low word is the text
		case useSpellBook: // Spell is the spell
			if testBit(mask16(rec.ClassMask), int(p.Class)) {
				s := int(rec.Spell) % 256
				if int(spellTable[s].Level)<<1 > int(p.Level) {
					showTextPages(menu, msgYouAreNotAdvanced)
				} else {
					setBit(p.KnownSpells[:], s, true)
					showTextPages(menu, msgSpellHasBeenScribed)
					removeItemShown(m, i)
				}
			} else {
				showTextPages(menu, msgYouAreNotEligible)
			}
		case useSwitchMapFlags: // on map Damage.Base, clear event flag Damage.Count and set Damage.Sides
			if mp := int(rec.Damage.Base); gameStateArg == mp {
				setBit(levelFlags[mp][:], int(rec.Damage.Count), false)
				setBit(scenarioHdr.LevelFlags[mp][:], int(rec.Damage.Count), false)
				setBit(levelFlags[mp][:], int(rec.Damage.Sides), true)
				setBit(scenarioHdr.LevelFlags[mp][:], int(rec.Damage.Sides), true)
				useItemChargeShown(m, i)
			}
		case useDirectionOnMap8:
			if gameStateArg == 8 {
				partyCounters[effectDirection].B = 1
			}
		case useReadWithItem388:
			if pickedIs(388) {
				showTextPages(menu, msgPersonalLog)
			}
		}
		return
	}
	if rec.Spell == 0 {
		return
	}
	power, s := int(rec.Spell)/256, int(rec.Spell)%256
	if spellTable[s].Target == spellTargetMember {
		if pvSpellTarget = chooseMember(menu, msgSheetUseOnWho, -1); pvSpellTarget == -1 {
			return
		}
	}
	castSpell(m, s, power, pvSpellTarget, sheet, menu)
	useItemChargeShown(m, i)
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
// power, its damage, bonuses, powers and resistances, if power*3 beats its IdentifyLevel.
func showItemInfo(menu *Window, m, i, power int) {
	p := &party[m]
	rec := itemRec(int(p.Items[i].ID))
	w := createWindow(20, 8, 20, 12, 25, borderNone, 0, false)
	fillWindow(w, 0, 1)
	f := func(x, y int, ch byte) { setFrameGlyph(w, x, y, ch) }
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
	putIconAt(w, 1, 1, rec.Icon+1)
	putIconAt(w, 18, 1, rec.Icon+1)
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
	case itemTypeWeaponS, itemTypeWeaponE, itemTypeWeaponT, itemTypeLauncher:
		primary, hands, skill, attackFlags, damage, specials, regen = true, true, true, true, true, true, true
	case itemTypeMissile:
		secondary, damage, specials = true, true, true
	case itemTypeMisc:
		acLine, acColor, resists, regen = true, 3, true, true
	case itemTypeHelmet, itemTypeBodyArmor, itemTypeLegArmor, itemTypeGauntlets, itemTypeBoots:
		acLine, typeName, resists, regen = true, true, true, true
	case itemTypeShield:
		acLine, resists, regen = true, true, true
	case itemTypePotion, itemTypePowder:
		masks, primary, spell, attackFlags = false, true, true, true
	case itemTypeScroll:
		masks, secondary, spell = false, true, true
	case 14, 15:
		secondary, attackFlags, regen = true, true, true
	}

	switch {
	case power == 0:
		centered(getMessage(int(rec.Type)+msgItemTypeBase), 3, 1)
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
		} else if spell && rec.Spell != 0 {
			centered(getMessage(int(rec.Spell)%256+msgSpellNameBase), 5, 4)
		}
		if skill {
			s, c := getMessage(msgUnusable), 9
			if testBit(mask16(rec.RaceMask), int(p.Race)) && testBit(mask16(rec.ClassMask), int(p.Class)) && testBit([]byte{rec.SexMask, rec.Flags}, int(p.Sex)) {
				s, c = getMessage(weaponSkill(rec.Icon)+msgSkillNameBase), 2
			}
			centered(s, 6, c)
		}
		if attackFlags {
			if rec.Type == itemTypeLauncher {
				setCursor(w, 3, 7)
				printString(w, getMessage(msgShoot), 5)
			} else {
				n := 0
				for b := 0; b < 8; b++ {
					if testBit([]byte{rec.HandFlags, rec.Unk73}, b) {
						setCursor(w, n/3<<3+3, n%3+7)
						printString(w, getMessage(b+msgAttackKindBase), 5)
						n++
					}
				}
			}
		}
		setCursor(w, 1, 10)
		if primary || secondary {
			s := getMessage(msgPS)
			putChar(w, byte(byteAt(s, 0)), map[bool]int{true: 6, false: 14}[primary])
			putChar(w, '/', 14)
			putChar(w, byte(byteAt(s, 1)), map[bool]int{true: 6, false: 14}[rec.Flags&recItemOffHand != 0])
		} else if acLine {
			printString(w, getMessage(msgAC), acColor+8)
			w.CurX++
			printString(w, strconv.Itoa(-int(int8(rec.AC))), acColor)
		}
		if hands {
			msg := 457
			if rec.Flags&recItemTwoHanded != 0 {
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

	case power*3 > int(rec.IdentifyLevel):
		row := 3
		if damage {
			setCursor(w, 1, 3)
			printString(w, getMessage(msgItemDamage), 14)
			a, b, c := strconv.Itoa(int(rec.Damage.Count)), strconv.Itoa(int(rec.Damage.Sides)), strconv.Itoa(int(rec.Damage.Base))
			setCursor(w, 19-(len(a)+len(b)+len(c)+2), 3)
			printString(w, a, 5)
			printString(w, getMessage(msgDiceD), 5)
			printString(w, b, 5)
			putChar(w, '+', 5)
			printString(w, c, 5)
			if th := int8(rec.ToHitPen); th != 0 {
				setCursor(w, 1, 4)
				printString(w, getMessage(msgTohit), 14)
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
			printString(w, getMessage(msgRegen), 14)
			setCursor(w, 17, row)
			if d >= 0 {
				putChar(w, '+', 5)
			}
			printString(w, strconv.Itoa(d), 5)
			row = 5
		}
		if rec.Curse != 0 {
			setCursor(w, 1, row)
			printString(w, getMessage(msgCurse), 3)
			setCursor(w, 15, row)
			printCipher(w, int(rec.Curse), 1)
			row++
		}
		if rec.Spell != 0 {
			setCursor(w, 1, row)
			printString(w, getMessage(int(rec.Spell)%256+msgSpellNameBase), 4)
			setCursor(w, 16, row)
			if p.Items[i].Qty == 0 {
				printString(w, getMessage(msgMax), 1)
			} else {
				printInt(w, int(p.Items[i].Qty), 3, 1)
			}
			row++
		}
		if row > 3 && row < 8 {
			row++
		}
		if rec.Power != 0 {
			setCursor(w, 1, row)
			printString(w, getMessage(msgSpecialPower), 7)
			setCursor(w, 15, row)
			printCipher(w, int(rec.Power), 1)
			row++
		}
		if specials {
			n := 0
			for b := 0; b < 8; b++ {
				if rec.HitEffects[b] > 0 {
					setCursor(w, n/3<<3+3, n%3+row)
					printString(w, getMessage(b+msgPowerNameBase), 5)
					n++
				}
			}
		} else if resists {
			for r := range realmCount {
				v := min((int(rec.SkillBonus[2*r+1])+int(rec.SkillBonus[2*r+2]))/2, 99)
				setCursor(w, r/3*9+1, r%3+row)
				printString(w, getMessage(r+msgRealmBase), realmColors[r])
				setCursor(w, r/3*9+7, r%3+row)
				printInt(w, v, 2, 1)
				putChar(w, '%', realmColors[r])
			}
		}

	default:
		s := getMessage(msgFailure)
		setCursor(w, 10-(len(s)+1)/2, 4)
		printString(w, s, 1)
	}
	resumeWindowDraw(w)
	fillWindow(menu, ' ', 3)
	deferWindowDraw(menu)
	s := getMessage(msgPressExitPadded)
	setCursor(menu, 20-(len(s)+1)/2, 1)
	printStringHi(menu, s, 3)
	resumeWindowDraw(menu)
	refreshWindow(w)
	refreshWindow(menu)
	waitEnterOrClick()
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
