package game

import (
	"strconv"

	"github.com/retro-ware/wiz6-go/internal/dbs"
)

// WMNPC trading: the shop, selling and giving to the NPC, pilfering, and the spells and
// items the party uses during the meeting.

// npcApplySpell (WMNPC 817e) is WDOPT's apply_spell plus the two spells aimed at the NPC:
// 50 charms it, 56 draws a rumour out of it.
func npcApplySpell(caster, s, power, target int) {
	c := &party[caster]
	switch s {
	case spellLockInNPC, spellMapInNPC: // the lock and map spells do nothing here
	case spellCharmNPC:
		if npcRec.Unk8D&npcCharmProof != 0 || random(100) > int(int16(power*10+(int(c.Level)-npcLevel())*5+25)) { // C int wraps
			npcNoticeName(1544, false)
		} else {
			npcNoticeName(1543, false)
			npcCharm += int16(rollDice(formatDice(power<<1, 6, power<<1)))
		}
		return
	case spellDrawRumor:
		var s string
		if npcRec.Unk8D&npcRumorProof != 0 || random(int(c.Level)/2+power*3) < npcLevel() {
			s = getMessage(1545)
		} else {
			rumor := npcRumor(1) // before message 1546, as the C
			s = replaceCharWithString(getMessage(1546), rumor, '^')
		}
		npcNotice(s, false)
		npcWait(messageDelay)
		return
	default:
		applySpellOutsideCombat(caster, s, power, target)
	}
	switch s {
	case 10, 15, 26, 46, 57, 51, 64, 77:
		npcDrawMemberStatus(target, npcShowGold)
	case 80, 81:
		npcDrawMemberStatus(caster, npcShowGold)
	}
}

// npcCastSpell (WMNPC 9edb): member m casts a spell from its book and pays for it.
func npcCastSpell(win *Window, m int) {
	fillWindow(win, ' ', 3)
	r := chooseSpellBook(m, npcMenuInput, func(msg int) int { npcTarget = npcChooseMember(msg, true); return npcTarget })
	pvBlinkState = -1
	if r == -1 {
		return
	}
	power, s := r/256, r%256
	sp := &spellTable[s]
	p := &party[m]
	p.SP[sp.Realm].Cur -= int16(int(sp.Cost) * power)
	npcApplySpell(m, s, power, npcTarget)
	if p.Stamina -= int16(random(power) + int(sp.Level) + power); p.Stamina < 0 {
		p.Stamina = 0
		p.Status[statusAsleep] = uint8(random(6) + 6)
	}
	npcDrawMemberStatus(m, npcShowGold)
}

// npcUseItem (WMNPC 9e1e): member m uses the spell of one of its items.
func npcUseItem(win *Window, m int) {
	slot := npcChooseMemberItem(win, m, 1516, 0)
	if slot == -1 {
		return
	}
	rec := itemRec(int(party[m].Items[slot].ID))
	if rec.Spell == 0 {
		return
	}
	power, s := int(rec.Spell)/256, int(rec.Spell)%256
	if spellTable[s].Target == 1 {
		refreshAllWindows()
		if npcTarget = npcChooseMember(2032, true); npcTarget == -1 {
			return
		}
	}
	useItemCharge(m, slot)
	npcApplySpell(m, s, power, npcTarget)
}

// --- Item lists

// npcDrawItemList (WMNPC 9827) shows five rows of member m's listed items around sel, the
// one at sel highlighted when hi.
func npcDrawItemList(win *Window, m, sel int, hi bool) {
	if dtLog != nil { // the difftest stubs it
		return
	}
	drawListFrame(win, false)
	deferWindowDraw(win)
	p := &party[m]
	for row := -2; row <= 2; row++ {
		if sel+row < 0 || sel+row >= len(npcItemSlots) {
			continue
		}
		setCursor(win, 2, row+2)
		slot := npcItemSlots[sel+row]
		it := p.Items[slot]
		stack := stackable[it.Type] == 1
		if hi && row == 0 {
			printString(win, npcItemNames[slot], 3)
			if stack {
				padToColumn(win, 15, 3)
				printInt(win, int(it.Qty), 3, 1)
			} else {
				padToColumn(win, 18, 3)
			}
		} else {
			printStringHi(win, npcItemNames[slot], 3)
			if stack {
				setCursor(win, 15, row+2)
				printInt(win, int(it.Qty), 3, 18)
			}
		}
		setCursor(win, 18, row+2)
		putCharHi(win, it.Icon+1, 4)
	}
	resumeWindowDraw(win)
	refreshWindow(win)
}

// npcAddItem (WMNPC 99e6) lists item slot of member m.
func npcAddItem(m, slot int) {
	rec := itemRec(int(party[m].Items[slot].ID))
	npcItemNames[slot] = cstr(rec.Name[:])
	npcItemSlots = append(npcItemSlots, slot)
}

// npcScrollList runs a five-row list beside an exit label at ex, ey of lw: draw shows it
// around sel, highlighted when not on the exit. The chosen index, or -1 for exit.
func npcScrollList(list, lw *Window, ex, ey int, exitHi, exitLo string, n, sel int, draw func(sel int, hi bool)) int {
	setClickRegion(0, list.X+2, list.Y+2, list.X+16, list.Y+2)
	setClickRegion(1, list.X+1, list.Y, list.X+1, list.Y)
	setClickRegion(2, list.X+1, list.Y+4, list.X+1, list.Y+4)
	setClickRegion(3, lw.X+byte(ex), lw.Y+byte(ey), lw.X+byte(ex+len(exitHi)-1), lw.Y+byte(ey))
	setClickRegion(4, clickRegionsEnd, 0, 0, 0)
	onExit := false
	for {
		setCursor(lw, ex, ey)
		if onExit {
			printString(lw, exitLo, -3)
		} else {
			printStringHi(lw, exitHi, 3)
		}
		draw(sel, !onExit)
		k := npcMenuInput()
		if menuClicked == 1 {
			switch k { // the click regions above
			case 0:
				k = npcKeySelect
			case 1:
				k = npcKeyUp
			case 2:
				k = npcKeyDown
			case 3:
				onExit, k = true, npcKeySelect
			}
		}
		switch k {
		case npcKeyLeft:
			onExit = true
		case npcKeyUp:
			if onExit {
				onExit = false
			} else if sel > 0 {
				sel--
			}
		case npcKeyRight:
			onExit = false
		case npcKeyDown:
			if onExit {
				onExit = false
			} else if sel+1 < n {
				sel++
			}
		}
		if k == npcKeySelect {
			break
		}
	}
	if onExit {
		return -1
	}
	return sel
}

// npcChooseMemberItem (WMNPC 9a42) lists member m's items and swag under prompt msg with
// the cursor on slot sel; the chosen slot or -1.
func npcChooseMemberItem(win *Window, m, msg, sel int) int {
	if dtLog != nil { // the difftest stubs it
		return 0
	}
	fillWindow(win, ' ', 3)
	list := createWindow(20, 19, 20, 5, 20, borderTopEdge, 0, false)
	fillWindow(list, ' ', 3)
	npcPrompt(replaceCharWithString(getMessage(1508), copyName(memberName(m)), '$'), 0)
	eraseCursorMark(bottomLineWin, false)
	deferWindowDraw(win)
	setCursor(win, 2, 1)
	printMessage(msg, win, 18)
	exit := getMessage(2035)
	setCursor(win, 7, 3)
	printStringHi(win, exit, 3)
	resumeWindowDraw(win)
	drawListFrame(list, false)
	p := &party[m]
	npcItemSlots = npcItemSlots[:0]
	at := 0
	for i := 0; i < int(p.ItemCount); i++ {
		npcAddItem(m, i)
		if i == sel {
			at = len(npcItemSlots) - 1
		}
	}
	for i := 0; i < int(p.SwagCount); i++ {
		npcAddItem(m, i+packSlots)
		if i+packSlots == sel {
			at = len(npcItemSlots) - 1
		}
	}
	if len(npcItemSlots) == 0 { // the C runs an empty list; nothing can be chosen
		destroyWindow(list, false)
		return -1
	}
	at = min(at, len(npcItemSlots)-1)
	refreshAllWindows()
	r := npcScrollList(list, win, 7, 3, exit, exit, len(npcItemSlots), at, func(sel int, hi bool) {
		npcDrawItemList(list, m, sel, hi)
	})
	destroyWindow(list, false)
	if r == -1 {
		return -1
	}
	return npcItemSlots[r]
}

// --- The shop

// npcListWares (WMNPC a206) lists the wares not yet sold.
func npcListWares() {
	npcWareList = npcWareList[:0]
	for i, w := range npcWares {
		if w.ID > 0 {
			npcWareList = append(npcWareList, i)
		}
	}
}

// npcPriceWares (WMNPC a247) reads the wares' records: a ware comes with its most charges,
// a stack priced by the piece, all at the NPC's buy percentage.
func npcPriceWares() {
	for _, i := range npcWareList {
		w := &npcWares[i]
		rec := itemRec(w.ID)
		w.Name = cstr(rec.Name[:])
		w.Type = rec.Type
		w.Qty = int(int16(int(rec.Charges.Count)*int(rec.Charges.Sides) + int(rec.Charges.Base))) // C int wraps
		price := rec.Price
		if stackable[w.Type] == 1 {
			price *= int32(w.Qty)
		}
		w.Price = int32(npcRec.Unk68[5]) * price / 100
		w.Icon = rec.Icon
		w.Class, w.Race, w.Sex = rec.ClassMask, rec.RaceMask, rec.SexMask
	}
}

// npcStockWares (WMNPC b805) rolls the NPC's wares from its treasure table (kind 2
// entries: items A.. with B tries of Chance% each).
func npcStockWares() {
	n := 0
	if npcRec.Treasure > 0 {
		t := npcRecord[TreasureRec](dbs.RecTreasure, int(npcRec.Treasure))
		for _, e := range t.Entries {
			if e.Kind != 2 || e.A <= 0 {
				continue
			}
			for ; n < 29 && e.B > 0; e.A, e.B = e.A+1, e.B-1 {
				if random(100) < int(e.Chance) {
					npcWares[n].ID = int(e.A)
					n++
				}
			}
		}
	}
	npcWares[n].ID = 0 // as the C, only an end mark: wares past it from an earlier stocking stay listed
	npcListWares()
	npcPriceWares()
}

// npcDrawWareList (WMNPC 9fcf) shows five rows of the wares around sel with what they are
// (or that member m cannot use them) and their price.
func npcDrawWareList(win *Window, m, sel int, hi bool) {
	if dtLog != nil { // the difftest stubs it
		return
	}
	drawListFrame(win, false)
	deferWindowDraw(win)
	p := &party[m]
	for row := -2; row <= 2; row++ {
		if sel+row < 0 || sel+row >= len(npcWareList) {
			continue
		}
		setCursor(win, 2, row+2)
		w := &npcWares[npcWareList[sel+row]]
		stack := stackable[w.Type] == 1
		if hi && row == 0 {
			printString(win, w.Name, 3)
			if stack {
				padToColumn(win, 15, 3)
				printInt(win, w.Qty, 3, 1)
			} else {
				padToColumn(win, 18, 3)
			}
		} else {
			printStringHi(win, w.Name, 3)
			if stack {
				setCursor(win, 15, row+2)
				printInt(win, w.Qty, 3, 18)
			}
		}
		setCursor(win, 19, row+2)
		putCharHi(win, w.Icon+1, 4)
		win.CurX++
		if w.Class>>p.Class&1 == 0 || w.Race>>p.Race&1 == 0 || w.Sex>>p.Sex&1 == 0 {
			printMessage(1567, win, 18)
		} else {
			printMessage(int(w.Type)+1550, win, 18)
		}
		win.CurX = 32
		printLong(win, w.Price, 7, 18)
	}
	resumeWindowDraw(win)
	refreshWindow(win)
}

// npcChooseWare (WMNPC a378) lists the wares for member m with the cursor on sel; the
// chosen index into npcWareList, or -1.
func npcChooseWare(win *Window, m, sel int) int {
	if dtLog != nil { // the difftest stubs it
		return 0
	}
	deferWindowDraw(mainWindow)
	eraseWindow(mainWindow, false)
	setCursor(mainWindow, 2, 0)
	printStringHi(mainWindow, getMessage(1509), 3)
	resumeWindowDraw(mainWindow)
	refreshWindow(mainWindow)
	refreshWindow(bottomLineWin)
	exitHi, exitLo := getMessage(1506), getMessage(1507)
	npcListWares()
	if len(npcWareList) == 0 {
		return -1
	}
	sel = min(sel, len(npcWareList)-1)
	return npcScrollList(win, mainWindow, 12, 0, exitHi, exitLo, len(npcWareList), sel, func(sel int, hi bool) {
		npcDrawWareList(win, m, sel, hi)
	})
}

// npcBuyWare (WMNPC a69b): member m buys ware i of the list.
func npcBuyWare(m, i int) {
	k := npcWareList[i]
	w := &npcWares[k]
	if party[m].Gold < w.Price {
		npcRefuse(1503)
		return
	}
	if npcGiveItemToMember(m, w.ID, w.Qty) == -1 {
		npcRefuse(1505)
		return
	}
	party[m].Gold -= w.Price
	id := w.ID
	w.ID = 0
	npcDrawMemberStatus(m, npcShowGold)
	npcReact(m, npcReactItemBought, id, 0, 0)
}

// npcShop (WMNPC a76f): member m buys until it leaves the list.
func npcShop(m int) {
	win := createWindow(0, 19, 40, 5, 25, borderNone, 0, false)
	fillWindow(win, ' ', 3)
	eraseWindow(bottomLineWin, false)
	for sel := 0; ; {
		if sel = npcChooseWare(win, m, sel); sel == -1 {
			break
		}
		npcBuyWare(m, sel)
	}
	eraseCursorMark(bottomLineWin, false)
	destroyWindow(win, false)
}

// npcSellItem (WMNPC a805) offers member m's item slot to the NPC at its sell percentage;
// the reaction script's result if sold, else 0.
func npcSellItem(win *Window, m, slot int) int {
	p := &party[m]
	it := p.Items[slot]
	id := int(it.ID)
	rec := itemRec(id)
	full := max(int(int16(int(rec.Charges.Count)*int(rec.Charges.Sides)+int(rec.Charges.Base))), 1) // C int wraps
	qty := max(int(it.Qty), 1)
	price := rec.Price
	if stackable[rec.Type] == 1 {
		price *= int32(qty)
	} else {
		price = price * 100 / int32(full) * int32(qty) / 100
	}
	price = int32(npcRec.Unk68[6]) * price / 100
	accepted := id == int(npcRec.Unk68[7]) || id == int(npcRec.Unk68[8])
	switch {
	case rec.Flags&itemImportant != 0 && !accepted:
		npcRefuse(1529)
		return 0
	case price <= 0 && id != int(npcRec.Unk68[7]):
		npcRefuse(1518)
		return 0
	case it.Flags&itemEquipped != 0:
		npcRefuse(1528)
		return 0
	}
	s := replaceCharWithString(getMessage(1525), itoa(int(price)), '$')
	fillWindow(win, ' ', 3)
	refreshWindow(win)
	setCursor(win, 10-(len(s)+1)/2, 1)
	printStringHi(win, s, 3)
	if menuSelectCancel(win, 1526, []int{1, 1, -1}, 6, 3, 5, 1, 5, 0, 1, npcMenuInput) != 0 {
		return 0
	}
	removeItem(m, slot)
	p.Gold += price
	npcDrawMemberStatus(m, npcShowGold)
	return npcReact(m, npcReactItemSold, id, 0, 0)
}

// npcSell (WMNPC aae4): member m offers items until it leaves; 1 if a sale ended it.
func npcSell(win *Window, m int) int {
	for slot := 0; ; {
		if slot = npcChooseMemberItem(win, m, 1517, slot); slot == -1 {
			return 0
		}
		if npcSellItem(win, m, slot) == 1 {
			return 1
		}
	}
}

// npcGiveGold (WMNPC ab2a): member m gives the NPC an amount of gold, then perhaps an item;
// both raise its attitude. The item's reaction script's result, else 0.
func npcGiveGold(win *Window, m int) int {
	p := &party[m]
	a := npcAttitude(m)
	level := int32(max(npcLevel(), 1))
	var n int32
	for {
		fillWindow(win, ' ', 3)
		deferWindowDraw(win)
		setCursor(win, 5, 1)
		printMessage(1581, win, 18)
		setCursor(win, 2, 3)
		printMessage(1582, win, 18)
		x, y := int(win.CurX), int(win.CurY)
		printString(win, "         ", 1)
		resumeWindowDraw(win)
		refreshAllWindows()
		setCursor(win, x, y)
		s := npcInputLine(win, 8, 5, inputDigits)
		n = 0
		if s == "" {
			break
		}
		v, _ := strconv.ParseInt(s, 10, 32)
		if n = int32(v); n >= 0 && n <= p.Gold {
			p.Gold -= n
			npcRec.Gold += n
			*a = int8(min(int16(int32(*a)+n/10/level), maxAttitude)) // (int) cast before the clamp
			break
		}
		playSound(soundBeep)
	}
	npcDrawMemberStatus(m, npcShowGold)
	npcReact(m, npcReactGoldGiven, int(int16(n/256)), uint8(n%256), 0)
	slot := npcChooseMemberItem(win, m, 1580, 0)
	if slot == -1 {
		return 0
	}
	it := p.Items[slot]
	if it.Flags&itemEquipped != 0 {
		npcRefuse(1528)
		return 0
	}
	id := int(it.ID)
	rec := itemRec(id)
	if rec.Flags&itemImportant != 0 && id != int(npcRec.Unk68[7]) && id != int(npcRec.Unk68[8]) {
		npcRefuse(1529)
		return 0
	}
	*a = int8(min(int16(int32(*a)+rec.Price/20/level), maxAttitude)) // (int) cast before the clamp
	removeItem(m, slot)
	npcDrawMemberStatus(m, npcShowGold)
	npcRefuse(1504)
	return npcReact(m, npcReactItemGiven, id, 0, 0)
}

// npcTrade's choices (messages 1520..).
const (
	tradeBuy = iota
	tradeSell
	tradeGive
	tradeDone
)

// npcTrade (WMNPC aeb5): member m buys, sells or gives until done, the panels showing gold.
func npcTrade(m int) {
	enabled := []int{1, 1, 1, 1, -1}
	if npcRec.Unk8D&npcNoBuySell != 0 {
		enabled[0], enabled[1] = 0, 0
	}
	win := createWindow(0, 19, 19, 5, 20, borderRightEdge, 0, false)
	title := replaceCharWithString(getMessage(1508), copyName(memberName(m)), '$')
	npcShowGold = 1
	for choice := tradeBuy; choice != tradeDone; {
		npcListWares()
		if len(npcWareList) == 0 {
			enabled[0] = 0
		}
		npcDrawMemberStatus(m, npcShowGold)
		fillWindow(win, ' ', 3)
		npcPrompt(title, 0)
		refreshAllWindows()
		choice = menuSelectCancel(win, 1520, enabled, 4, 1, 7, 2, 5, 0, countEnabledItems(enabled, choice), npcMenuInput)
		switch choice {
		case tradeBuy:
			npcShop(m)
		case tradeSell:
			if npcSell(win, m) == 1 {
				choice = tradeDone
			}
		case tradeGive:
			if npcGiveGold(win, m) == 1 {
				choice = tradeDone
			}
		}
	}
	destroyWindow(win, false)
	npcShowGold = 0
	npcDrawMemberStatus(m, npcShowGold)
}

// npcPilfer (WMNPC b047): member m tries to steal gold or a ware. Thieves and rangers,
// bards and ninjas are better at it; each try makes the next harder. Caught, the
// member's attitude drops.
func npcPilfer(m int) {
	p := &party[m]
	chance := (int(p.Level)-npcLevel())*5 + int(p.Skills[skillPilfer])/2
	switch p.Class {
	case classThief:
		chance += int(p.Stats[statDexterity]) << 1
	case classRanger, classBard, classNinja:
		chance += int(p.Stats[statDexterity])
	}
	chance = min(max(int(int16(int(p.Stats[statPersonality])+int(int16(int(*npcAttitude(m))+int(npcCharm)))/4+chance-10)), 0), 95) // C int wraps
	roll := random(100) + npcPilfered
	npcPilfered += random(10) + 5
	npcNotice(replaceCharWithString(getMessage(int(p.Sex)+1570), memberName(m), '^'), false)
	var msg int
	switch {
	case roll < chance:
		npcListWares()
		if len(npcWareList) == 0 || random(100) < 20 {
			n := int32(random(100) * npcLevel())
			n = min(n, npcRec.Gold)
			npcRec.Gold -= n
			p.Gold += n
			msg = 1574
		} else if w := &npcWares[npcWareList[random(len(npcWareList))]]; npcGiveItemToMember(m, w.ID, w.Qty) == -1 {
			msg = 1575
		} else {
			msg = 1573
			w.ID = 0
		}
	case chance<<1 >= roll && roll < 95:
		msg = 1572
	default:
		npcNotice(replaceCharWithString(getMessage(1576), memberName(m), '^'), true)
		a := npcAttitude(m)
		*a = int8(max(int(*a)-(random(25)+25), minAttitude))
		npcReact(m, npcReactPilferCaught, 0, 0, 0)
		return
	}
	npcNotice(replaceCharWithString(getMessage(msg), memberName(m), '^'), true)
}
